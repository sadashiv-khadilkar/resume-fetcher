# Resume Fetcher

A personal CLI tool that finds candidates on Naukri and LinkedIn matching a given job description, using the operator's own paid Naukri Resdex and LinkedIn Recruiter seats.

## Language

**JD**:
The job description text that drives a Fetch Run — the source of both the native-search filters and the input to ranking. Supplied as Markdown initially.
_Avoid_: Job posting, requirement doc

**Candidate**:
A person surfaced by a search on Naukri or LinkedIn for a given JD. Holds one Profile per Source they were found on (merged into one Candidate when the same person is found on both), and optionally a Resume.
_Avoid_: Result, lead, applicant

**Profile**:
Structured data about a Candidate (name, contact info, skills, experience, education) scraped from their Naukri Resdex or LinkedIn page. Always available for a Candidate.
_Avoid_: Resume, CV, record

**Resume**:
An actual downloadable file (PDF/DOC) representing a Candidate's CV — from a Naukri Resdex resume download or a LinkedIn profile PDF export. Not every Candidate has one available: LinkedIn only offers it per-profile and subject to that person's settings; Naukri downloads consume Resdex credits.
_Avoid_: Profile, CV

**Match Score**:
The output of the LLM re-rank pass: a Candidate's fit against the full JD text, computed after the native-search filter pass has already narrowed the pool.
_Avoid_: Rank, relevance

**Fetch Run**:
A single execution of the tool against one JD, producing its own output folder (Candidates, Profiles, and any downloaded Resumes from that run).
_Avoid_: Session, job (ambiguous with the job description itself)
