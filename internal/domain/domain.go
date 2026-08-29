// Package domain holds the shared vocabulary from CONTEXT.md: JD, Candidate,
// Profile, Resume, and Match Score, used across the pipeline, platform, and
// llmprovider packages.
package domain

// Source identifies which platform a Profile was found on.
type Source string

const (
	SourceNaukri   Source = "naukri"
	SourceLinkedIn Source = "linkedin"
)

// Filters are the native search filters extracted from a JD.
type Filters struct {
	Skills        []string
	MinExperience int
	MaxExperience int
	Location      string
	NoticePeriod  string
}

// Profile is structured data about a Candidate as seen on one Source.
type Profile struct {
	ExternalID string
	Source     Source
	Name       string
	Email      string
	Phone      string
	Company    string
	Title      string
	Skills     []string
	Experience int
	Location   string
	Education  string
	URL        string
}

// ResumeFile is the result of a resume-download attempt for a Profile.
type ResumeFile struct {
	Available bool
	Data      []byte
	Ext       string
}

// Candidate is a person surfaced by a search, merged across every Source
// they were found on.
type Candidate struct {
	ID          string
	Profiles    []Profile
	FuzzyMerged bool
	MatchScore  float64
	ResumePath  string
	ResumeNote  string
}

// Match is one Candidate's Match Score from an LLMProvider ranking pass.
type Match struct {
	CandidateID string
	Score       float64
}
