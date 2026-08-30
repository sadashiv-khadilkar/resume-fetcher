# 02: Real Claude LLMProvider

**What to build:** Replace the fake `LLMProvider` with a real Claude API-backed implementation that extracts search filters from a JD and computes a Match Score ranking for a set of Profiles. Demoed against ticket 01's fake `PlatformClient`s, so this ticket verifies genuine LLM behavior without needing real scraping in place yet.

**Blocked by:** 01

**Status:** done

- [x] A Claude API implementation of `LLMProvider` exists, selected via config/env var (e.g. `LLM_PROVIDER=claude`), alongside the fake from ticket 01
- [x] `ExtractFilters` returns real, structured filters (skills, experience range, location, notice period, etc.) derived from an actual JD's text
- [x] `Rank` returns a genuine Match Score per Candidate reflecting fit against the full JD text, not just keyword overlap (per-Candidate, not per-Profile — matches the `LLMProvider` interface as it landed in ticket 01)
- [x] A failed or rate-limited Claude API call is retried with backoff a bounded number of times
- [x] If retries are exhausted, the raw (unranked) candidate pool is persisted to disk and the run stops cleanly, rather than losing already-fetched data
- [x] `ExtractFilters` and `Rank` are tested against the real API only for a smoke-level check; the bulk of ranking/extraction-consuming logic remains tested via the fake from ticket 01

## Implementation notes

- `internal/llmprovider/claude`: real `Provider` using the official `github.com/anthropics/anthropic-sdk-go`. Both `ExtractFilters` and `Rank` force a single tool call (`extract_filters` / `submit_match_scores`) so the response is always parseable, rather than parsing free text.
- Retry/backoff is the SDK's own (`option.WithMaxRetries`, set to 4) rather than hand-rolled — the SDK already retries 429/5xx with exponential backoff.
- Model defaults to `claude-opus-5`, overridable via `ANTHROPIC_MODEL`. Provider selection is `LLM_PROVIDER=claude|fake` (case-insensitive), defaulting to `fake` so existing runs/tests stay network-free unless the operator opts in.
- On a `Rank` failure — either `LLMProvider.Rank` itself erroring, or it succeeding but omitting a Match Score for some candidate — `pipeline.Orchestrator.Run` now returns a `*pipeline.RankFailedError` carrying the raw unranked pool; `internal/cli`'s `writePartialOnRankFailure` persists it via the existing output writer and prints a clear message before returning the error.
- `MaxTokens` on both Claude calls is 16000: `claude-opus-5` runs adaptive thinking by default (thinking tokens count against `MaxTokens`), so a small budget sized only for the JSON output risked truncating the response before the forced tool call was ever emitted.
- Smoke tests (`internal/llmprovider/claude/claude_test.go`) skip via `t.Skip` when `ANTHROPIC_API_KEY` is unset — this sandbox has no key configured, so they were verified to skip cleanly but not run live. Worth an operator running them once against a real key before relying on this provider for a real Fetch Run.
- Config (`LLM_PROVIDER`, `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`) is loaded from a gitignored `.env` file via `godotenv` at the CLI entrypoint, documented in `.env.example` (see `ADR-0004`).
- Identity-linked Anthropic API keys (issued via org SSO) are rejected with `anthropic-workspace-id is required` until `ANTHROPIC_WORKSPACE_ID` is set — discovered while running these smoke tests against a real key. The Go SDK reads it from the env var automatically; it's documented in `.env.example`.
