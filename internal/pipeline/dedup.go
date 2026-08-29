package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"

	"resumefetcher/internal/domain"
)

var nonDigits = regexp.MustCompile(`\D+`)

// MergeProfiles merges Profiles found across platforms into Candidates.
// Two Profiles merge when they share an exact email or phone number; failing
// that, they merge on a fuzzy match (same normalized name and company), which
// is flagged via Candidate.FuzzyMerged so it can be reviewed rather than
// silently trusted.
func MergeProfiles(profiles []domain.Profile) []domain.Candidate {
	var groups [][]domain.Profile
	fuzzyGroup := map[int]bool{}

	for _, p := range profiles {
		if idx, ok := findExactGroup(groups, p); ok {
			groups[idx] = append(groups[idx], p)
			continue
		}
		if idx, ok := findFuzzyGroup(groups, p); ok {
			groups[idx] = append(groups[idx], p)
			fuzzyGroup[idx] = true
			continue
		}
		groups = append(groups, []domain.Profile{p})
	}

	candidates := make([]domain.Candidate, len(groups))
	for i, g := range groups {
		candidates[i] = domain.Candidate{
			ID:          candidateID(g),
			Profiles:    g,
			FuzzyMerged: fuzzyGroup[i],
		}
	}
	return candidates
}

func findExactGroup(groups [][]domain.Profile, p domain.Profile) (int, bool) {
	for i, g := range groups {
		for _, existing := range g {
			if exactMatch(existing, p) {
				return i, true
			}
		}
	}
	return 0, false
}

func findFuzzyGroup(groups [][]domain.Profile, p domain.Profile) (int, bool) {
	for i, g := range groups {
		for _, existing := range g {
			if fuzzyMatch(existing, p) {
				return i, true
			}
		}
	}
	return 0, false
}

func exactMatch(a, b domain.Profile) bool {
	if email := normalizeEmail(a.Email); email != "" && email == normalizeEmail(b.Email) {
		return true
	}
	if phone := normalizePhone(a.Phone); phone != "" && phone == normalizePhone(b.Phone) {
		return true
	}
	return false
}

func fuzzyMatch(a, b domain.Profile) bool {
	name := normalizeText(a.Name)
	company := normalizeText(a.Company)
	if name == "" || company == "" {
		return false
	}
	return name == normalizeText(b.Name) && company == normalizeText(b.Company)
}

// normalizeEmail returns "" for anything that isn't a plausible email, so a
// masked or placeholder value (e.g. "N/A") scraped from a platform never
// counts as an exact match. See normalizePhone for the equivalent guard.
func normalizeEmail(s string) string {
	e := normalizeText(s)
	if !strings.Contains(e, "@") {
		return ""
	}
	return e
}

// normalizePhone returns "" unless s reduces to a full 10-digit number, so a
// masked value (e.g. "*******3210") never counts as an exact match.
func normalizePhone(s string) string {
	digits := nonDigits.ReplaceAllString(s, "")
	if len(digits) < 10 {
		return ""
	}
	return digits[len(digits)-10:]
}

func normalizeText(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// candidateID is deterministic in the profiles it merges, so re-running the
// same search produces stable Candidate IDs.
func candidateID(profiles []domain.Profile) string {
	keys := make([]string, len(profiles))
	for i, p := range profiles {
		keys[i] = string(p.Source) + ":" + p.ExternalID
	}
	sort.Strings(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "|")))
	return hex.EncodeToString(sum[:])[:12]
}
