package pipeline_test

import (
	"testing"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/pipeline"
)

func TestMergeProfiles(t *testing.T) {
	tests := []struct {
		name       string
		profiles   []domain.Profile
		wantGroups int
	}{
		{
			name: "no overlap stays separate",
			profiles: []domain.Profile{
				{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Email: "asha@example.com"},
				{ExternalID: "l1", Source: domain.SourceLinkedIn, Name: "Vikram Shah", Email: "vikram@example.com"},
			},
			wantGroups: 2,
		},
		{
			name: "exact email match merges",
			profiles: []domain.Profile{
				{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Email: "Asha@Example.com"},
				{ExternalID: "l1", Source: domain.SourceLinkedIn, Name: "Asha R.", Email: "asha@example.com"},
			},
			wantGroups: 1,
		},
		{
			name: "exact phone match merges",
			profiles: []domain.Profile{
				{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Phone: "+91 98765 43210"},
				{ExternalID: "l1", Source: domain.SourceLinkedIn, Name: "Asha R.", Phone: "9876543210"},
			},
			wantGroups: 1,
		},
		{
			name: "fuzzy name+company match merges and is flagged",
			profiles: []domain.Profile{
				{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Company: "Acme Corp"},
				{ExternalID: "l1", Source: domain.SourceLinkedIn, Name: "asha rao", Company: "ACME CORP"},
			},
			wantGroups: 1,
		},
		{
			name: "same name different company stays separate",
			profiles: []domain.Profile{
				{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Company: "Acme Corp"},
				{ExternalID: "l1", Source: domain.SourceLinkedIn, Name: "Asha Rao", Company: "Globex Inc"},
			},
			wantGroups: 2,
		},
		{
			name: "masked phone numbers do not false-merge",
			profiles: []domain.Profile{
				{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Phone: "*******3210"},
				{ExternalID: "l1", Source: domain.SourceLinkedIn, Name: "Priya Singh", Phone: "*******3210"},
			},
			wantGroups: 2,
		},
		{
			name: "placeholder email does not false-merge",
			profiles: []domain.Profile{
				{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Email: "N/A"},
				{ExternalID: "l1", Source: domain.SourceLinkedIn, Name: "Priya Singh", Email: "N/A"},
			},
			wantGroups: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pipeline.MergeProfiles(tt.profiles)
			if len(got) != tt.wantGroups {
				t.Fatalf("MergeProfiles() produced %d candidates, want %d", len(got), tt.wantGroups)
			}
		})
	}
}

func TestMergeProfiles_FlagsFuzzyMatchOnly(t *testing.T) {
	profiles := []domain.Profile{
		{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Company: "Acme Corp"},
		{ExternalID: "l1", Source: domain.SourceLinkedIn, Name: "asha rao", Company: "acme corp"},
	}

	got := pipeline.MergeProfiles(profiles)
	if len(got) != 1 {
		t.Fatalf("MergeProfiles() produced %d candidates, want 1", len(got))
	}
	if !got[0].FuzzyMerged {
		t.Errorf("FuzzyMerged = false, want true for a name+company-only match")
	}

	exactMatch := []domain.Profile{
		{ExternalID: "n2", Source: domain.SourceNaukri, Name: "Ravi Kumar", Email: "ravi@example.com"},
		{ExternalID: "l2", Source: domain.SourceLinkedIn, Name: "Ravi K.", Email: "ravi@example.com"},
	}
	got2 := pipeline.MergeProfiles(exactMatch)
	if len(got2) != 1 {
		t.Fatalf("MergeProfiles() produced %d candidates, want 1", len(got2))
	}
	if got2[0].FuzzyMerged {
		t.Errorf("FuzzyMerged = true, want false for an exact email match")
	}
}

func TestMergeProfiles_DeterministicID(t *testing.T) {
	profiles := []domain.Profile{
		{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Email: "asha@example.com"},
	}

	first := pipeline.MergeProfiles(profiles)
	second := pipeline.MergeProfiles(profiles)

	if first[0].ID == "" {
		t.Fatal("candidate ID must not be empty")
	}
	if first[0].ID != second[0].ID {
		t.Errorf("candidate ID is not deterministic: %q != %q", first[0].ID, second[0].ID)
	}
}
