package reviewer_test

import (
	"bytes"
	"strings"
	"testing"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/reviewer"
)

func TestStdin_AcceptsFiltersOnEnter(t *testing.T) {
	in := strings.NewReader("\n")
	var out bytes.Buffer

	s := reviewer.NewStdin(in, &out)
	got, err := s.Review(domain.Filters{Location: "Pune", Skills: []string{"Go"}})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if got.Location != "Pune" {
		t.Errorf("Location = %q, want unchanged %q", got.Location, "Pune")
	}
	if out.Len() == 0 {
		t.Error("expected the extracted filters to be printed")
	}
}

func TestStdin_EditsFieldsOnReject(t *testing.T) {
	// "n" rejects, then one line per prompted field: Skills, MinExperience,
	// MaxExperience, Location, NoticePeriod. Blank lines keep the old value.
	in := strings.NewReader("n\nGo, Rust\n\n\nBangalore\n\n")
	var out bytes.Buffer

	s := reviewer.NewStdin(in, &out)
	got, err := s.Review(domain.Filters{
		Skills:        []string{"Java"},
		MinExperience: 2,
		MaxExperience: 5,
		Location:      "Pune",
		NoticePeriod:  "30 days",
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if want := []string{"Go", "Rust"}; !equalStrings(got.Skills, want) {
		t.Errorf("Skills = %v, want %v", got.Skills, want)
	}
	if got.MinExperience != 2 {
		t.Errorf("MinExperience = %d, want unchanged 2 (blank input)", got.MinExperience)
	}
	if got.Location != "Bangalore" {
		t.Errorf("Location = %q, want %q", got.Location, "Bangalore")
	}
	if got.NoticePeriod != "30 days" {
		t.Errorf("NoticePeriod = %q, want unchanged %q (blank input)", got.NoticePeriod, "30 days")
	}
}

func TestStdin_ReportsInvalidIntAndKeepsOldValue(t *testing.T) {
	in := strings.NewReader("n\n\nfive\n\n\n\n")
	var out bytes.Buffer

	s := reviewer.NewStdin(in, &out)
	got, err := s.Review(domain.Filters{MinExperience: 2, MaxExperience: 5})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if got.MinExperience != 2 {
		t.Errorf("MinExperience = %d, want unchanged 2", got.MinExperience)
	}
	if !strings.Contains(out.String(), "not a number") {
		t.Errorf("expected the operator to be told 'five' wasn't a valid number, got:\n%s", out.String())
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
