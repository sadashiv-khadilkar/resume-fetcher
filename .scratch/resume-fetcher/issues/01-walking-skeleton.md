# 01: Walking skeleton — full pipeline with fakes

**What to build:** The `fetch --jd <file>` command runs the complete Resume Fetcher pipeline end-to-end — extract filters from the JD, prompt the operator to confirm/edit them, search both platforms, merge/dedupe candidates, rank them, select the top-N for resume download, and write the output — using fake `PlatformClient` and `LLMProvider` implementations. No network or browser calls happen in this ticket; it proves the orchestration, dedup, ranking-integration, and output logic all work together.

**Blocked by:** None (can start immediately)

**Status:** done

- [x] `PlatformClient` interface exists with `Login`, `Search(filters)`, `DownloadResume(profileID)`; two fake implementations exist for tests/demo
- [x] `LLMProvider` interface exists with `ExtractFilters(jd)`, `Rank(jd, profiles)`; a fake implementation exists for tests/demo
- [x] `fetch --jd <file>` reads a Markdown JD, calls `ExtractFilters`, and prints the extracted filters for the operator to confirm or edit before proceeding
- [x] The orchestrator calls `Search` on both fake `PlatformClient`s, capping the raw pool per platform (e.g. top 50)
- [x] Candidates appearing in both platforms' results are merged: exact email/phone match merges automatically; name+company fuzzy match merges but is flagged in the output
- [x] The orchestrator calls `Rank` and keeps only the top 20-50 Candidates by Match Score
- [x] `DownloadResume` is called only for the top N (default 10, configurable) ranked Candidates
- [x] Output is written as one folder per Fetch Run: a JSON file of Candidates/Profiles (including any flagged fuzzy merges) plus a `resumes/` subfolder
- [x] A summary table (name, Match Score, source) is printed to the terminal after a run completes
- [x] Orchestrator, dedup/merge, top-N selection, and output-writer logic are covered by tests using the fake `PlatformClient`/`LLMProvider` implementations — no real network or browser calls in any test

## Code review follow-ups deferred to later tickets

`/code-review` flagged that a `Search` or `DownloadResume` error currently aborts the whole run, discarding already-fetched data. Left as-is here since ticket 01's fakes never error; ticket 02 already owns LLM-failure retry/persist behavior and tickets 04/05/07/08 already own CAPTCHA/rate-limit pause-and-resume for real search/download — the fix belongs there, once the real recovery shape is known.
