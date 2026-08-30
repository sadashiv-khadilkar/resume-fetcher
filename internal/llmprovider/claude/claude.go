// Package claude implements llmprovider.Provider against the real Claude
// API (ticket 02). Retries for failed/rate-limited requests are handled by
// the SDK's own backoff (option.WithMaxRetries) rather than reimplemented
// here; when retries are exhausted, ExtractFilters/Rank simply return the
// SDK's error, and it's the pipeline package's job (RankFailedError) to
// preserve already-fetched data on a Rank failure.
package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/llmprovider"
)

var _ llmprovider.Provider = (*Provider)(nil)

// DefaultModel is used when New is given an empty model string.
const DefaultModel = "claude-opus-5"

// DefaultMaxRetries bounds how many times the SDK retries a failed or
// rate-limited request before giving up.
const DefaultMaxRetries = 4

// Provider is a real, Claude API-backed llmprovider.Provider.
// ANTHROPIC_API_KEY is read from the environment by the SDK; no key is
// handled or stored by this package directly.
type Provider struct {
	client anthropic.Client
	model  string
}

// New builds a Provider. model overrides DefaultModel when non-empty.
func New(model string) *Provider {
	if model == "" {
		model = DefaultModel
	}
	return &Provider{
		client: anthropic.NewClient(option.WithMaxRetries(DefaultMaxRetries)),
		model:  model,
	}
}

type extractedFilters struct {
	Skills        []string `json:"skills"`
	MinExperience int      `json:"min_experience"`
	MaxExperience int      `json:"max_experience"`
	Location      string   `json:"location"`
	NoticePeriod  string   `json:"notice_period"`
}

// ExtractFilters asks Claude to turn a JD's free text into structured
// native-search Filters, via a forced tool call so the response is always
// parseable.
func (p *Provider) ExtractFilters(jd string) (domain.Filters, error) {
	tool := anthropic.ToolParam{
		Name:        "extract_filters",
		Description: anthropic.String("Submit the native recruiter-search filters extracted from the JD: skills, an experience range in years, location, and notice period. Leave a field at its zero value if the JD doesn't specify it."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"skills":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"min_experience": map[string]any{"type": "integer"},
				"max_experience": map[string]any{"type": "integer"},
				"location":       map[string]any{"type": "string"},
				"notice_period":  map[string]any{"type": "string"},
			},
			Required: []string{"skills", "min_experience", "max_experience", "location", "notice_period"},
		},
	}

	resp, err := p.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model: p.model,
		// Generous relative to the small JSON output: claude-opus-5 runs
		// adaptive thinking by default (thinking tokens count against
		// MaxTokens), so an undersized budget can be exhausted before the
		// forced tool call is ever emitted.
		MaxTokens: 16000,
		System: []anthropic.TextBlockParam{{
			Text: "You extract native recruiter-search filters from a job description. Call extract_filters exactly once with your best-effort structured filters.",
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(jd)),
		},
		Tools:      []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceParamOfTool("extract_filters"),
	})
	if err != nil {
		return domain.Filters{}, fmt.Errorf("claude extract filters: %w", err)
	}

	for _, block := range resp.Content {
		tu, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok {
			continue
		}
		var in extractedFilters
		if err := json.Unmarshal(tu.Input, &in); err != nil {
			return domain.Filters{}, fmt.Errorf("claude extract filters: parse tool input: %w", err)
		}
		return domain.Filters{
			Skills:        in.Skills,
			MinExperience: in.MinExperience,
			MaxExperience: in.MaxExperience,
			Location:      in.Location,
			NoticePeriod:  in.NoticePeriod,
		}, nil
	}
	return domain.Filters{}, fmt.Errorf("claude extract filters: no tool_use block in response")
}

// candidateSummary is the slim, prompt-facing view of a Candidate sent to
// Claude for ranking - not domain.Candidate itself, to keep the prompt
// compact and stable regardless of internal field changes.
type candidateSummary struct {
	CandidateID string   `json:"candidate_id"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Company     string   `json:"company"`
	Skills      []string `json:"skills"`
	Experience  int      `json:"experience_years"`
	Location    string   `json:"location"`
	Education   string   `json:"education"`
}

type matchScores struct {
	Matches []struct {
		CandidateID string  `json:"candidate_id"`
		Score       float64 `json:"score"`
	} `json:"matches"`
}

// Rank asks Claude to score every Candidate's fit against the full JD text,
// via a forced tool call so the response is always parseable. It's the
// caller's (pipeline's) job to check every Candidate got a Match Score back.
func (p *Provider) Rank(jd string, candidates []domain.Candidate) ([]domain.Match, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	summaries := make([]candidateSummary, len(candidates))
	for i, c := range candidates {
		primary := c.Profiles[0]
		summaries[i] = candidateSummary{
			CandidateID: c.ID,
			Name:        primary.Name,
			Title:       primary.Title,
			Company:     primary.Company,
			Skills:      primary.Skills,
			Experience:  primary.Experience,
			Location:    primary.Location,
			Education:   primary.Education,
		}
	}
	summariesJSON, err := json.Marshal(summaries)
	if err != nil {
		return nil, fmt.Errorf("claude rank: marshal candidates: %w", err)
	}

	tool := anthropic.ToolParam{
		Name:        "submit_match_scores",
		Description: anthropic.String("Submit a Match Score (0-100) for every candidate_id listed, reflecting fit against the full JD text rather than keyword overlap. Include every candidate_id exactly once."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"matches": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"candidate_id": map[string]any{"type": "string"},
							"score":        map[string]any{"type": "number"},
						},
						"required": []string{"candidate_id", "score"},
					},
				},
			},
			Required: []string{"matches"},
		},
	}

	resp, err := p.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model: p.model,
		// See the MaxTokens comment in ExtractFilters: adaptive thinking over
		// a full batch of candidates needs real headroom, not just enough
		// for the final JSON.
		MaxTokens: 16000,
		System: []anthropic.TextBlockParam{{
			Text: "You rank Candidates against a job description's full text, not just keyword overlap. Call submit_match_scores exactly once with a Match Score for every candidate_id given.",
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				fmt.Sprintf("Job description:\n%s\n\nCandidates (JSON):\n%s", jd, summariesJSON),
			)),
		},
		Tools:      []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceParamOfTool("submit_match_scores"),
	})
	if err != nil {
		return nil, fmt.Errorf("claude rank: %w", err)
	}

	for _, block := range resp.Content {
		tu, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok {
			continue
		}
		var in matchScores
		if err := json.Unmarshal(tu.Input, &in); err != nil {
			return nil, fmt.Errorf("claude rank: parse tool input: %w", err)
		}
		matches := make([]domain.Match, len(in.Matches))
		for i, m := range in.Matches {
			matches[i] = domain.Match{CandidateID: m.CandidateID, Score: m.Score}
		}
		return matches, nil
	}
	return nil, fmt.Errorf("claude rank: no tool_use block in response")
}
