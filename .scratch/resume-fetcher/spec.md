# Resume Fetcher

Status: ready-for-agent

## Problem Statement

Sourcing candidates on Naukri and LinkedIn for a given JD is a fully manual, repeated-by-hand process: log into each site separately, hand-translate the JD into each platform's search filters, scroll through results, open individual profiles to judge fit, and separately download resumes. Results end up scattered across two disconnected accounts with no unified, JD-driven sense of who is actually the best match.

## Solution

A personal, local Go CLI tool that, given a JD, uses the operator's own already-paid Naukri Resdex and LinkedIn Recruiter accounts (logged in once, session persisted across runs) to: auto-derive native search filters from the JD (with a quick confirm/edit step), run each platform's native search, merge/dedupe candidates seen on both platforms into a single Candidate list, use an LLM to compute a Match Score ranking each Candidate against the full JD text, download resumes only for the top-ranked Candidates to conserve Resdex credits, and write everything to a per-run output folder plus a terminal summary table — pausing for the operator to resolve any CAPTCHA/2FA challenge that comes up mid-run.

## User Stories

**Login & session**

1. As the operator, I want to log into Naukri and LinkedIn myself in a visible browser window, so that my password is never stored by the tool.
2. As the operator, I want my login session to persist across runs, so that I don't have to log in (or solve 2FA) every time I run a fetch.
3. As the operator, I want to re-run `login` for a single platform, so that I can restore an expired session on one platform without touching the other's.

**JD & filters**

4. As the operator, I want to supply a JD as a Markdown file, so that I can reuse JDs I already have written.
5. As the operator, I want the tool to automatically extract native search filters (skills, experience range, location, notice period, etc.) from my JD, so that I don't have to manually translate the JD into each platform's search UI.
6. As the operator, I want to review and edit the auto-extracted filters before the search runs, so that a bad auto-extraction doesn't silently tank the whole candidate pool.
7. As the operator, I want to reject an auto-extracted filter set and have the tool re-extract or let me hand-edit it, so that I can correct mistakes before any search runs.

**Search & profile fetch**

8. As the operator, I want the tool to run the confirmed filters as a native search on both Naukri Resdex and LinkedIn Recruiter, so that I get results each platform itself considers relevant.
9. As the operator, I want the raw result pool capped per platform (e.g. top 50), so that the automation footprint and downstream LLM cost stay bounded.
10. As the operator, I want structured Profile data (name, contact info, skills, experience, education) fetched for each raw result, so that I have enough information to judge fit.

**Dedup**

11. As the operator, I want candidates appearing on both Naukri and LinkedIn merged into a single Candidate entry, so that I don't review the same person twice.
12. As the operator, I want exact email/phone matches merged automatically, so that obvious duplicates need no input from me.
13. As the operator, I want fuzzy matches (name + current company) flagged in the output, so that I can visually confirm a merge rather than silently trust a possible false positive.

**Ranking**

14. As the operator, I want the pooled candidates ranked by an LLM-computed Match Score against the full JD text, so that ordering reflects genuine JD fit, not just keyword hits.
15. As the operator, I want only the top 20-50 ranked candidates kept in the final output, so that I review a manageable shortlist rather than the full raw pool.
16. As the operator, I want the LLM provider swappable (Claude API today, OpenAI-compatible or local LLM later), so that I'm not locked into a single vendor.
17. As the operator, I want to select the active LLM provider via a config file/env var, so that switching providers doesn't require a code change.

**Resume download**

18. As the operator, I want resumes auto-downloaded only for the top N ranked candidates (default 10, configurable), so that I don't burn all my Resdex credits or add unnecessary automation risk for candidates I won't seriously consider.
19. As the operator, I want a downloaded resume saved alongside its Candidate's structured data, so that I can open a CV directly without revisiting the platform.
20. As the operator, I want to know when a resume couldn't be obtained for a top-ranked candidate, so that a missing file isn't mistaken for a bug.

**Output & UX**

21. As the operator, I want each fetch to produce its own Fetch Run output folder (a JSON file of Candidates/Profiles plus a `resumes/` subfolder), so that results from different JDs never mix.
22. As the operator, I want a summary table (name, Match Score, source) printed to the terminal when a run finishes, so that I get immediate feedback without opening a file.
23. As the operator, I want a `runs` command listing past Fetch Runs, so that I can find and reopen earlier results.

**Resilience**

24. As the operator, I want the browser to pause visibly and wait for me when it hits a CAPTCHA or 2FA prompt mid-run, so that I can resolve it myself without losing progress.
25. As the operator, I want the run to continue automatically once I resolve a CAPTCHA/2FA challenge, so that I don't have to restart the whole fetch.
26. As the operator, I want a failed/rate-limited LLM call retried with backoff, so that transient API issues don't abort an otherwise-successful run.
27. As the operator, I want the raw (unranked) candidate pool saved to disk if LLM ranking ultimately fails after retries, so that the riskiest-to-reacquire data isn't lost.

**CLI**

28. As the operator, I want a `login <platform>` command to capture a session for one platform at a time, so that I can manage Naukri and LinkedIn sessions independently.
29. As the operator, I want a `fetch --jd <file> [--source naukri|linkedin|both] [--top-n N]` command that runs the full pipeline against a given JD file, so that one command produces a finished shortlist, optionally scoped to a single platform.

## Implementation Decisions

- **Language/runtime**: Go (`ADR-0002`).
- **Browser automation**: `go-rod` + `go-rod/stealth`, driven non-headless so the operator can see and resolve CAPTCHA/2FA (`ADR-0002`).
- **Access approach**: automation against the operator's own logged-in Naukri Resdex and LinkedIn Recruiter sessions, ToS/suspension risk knowingly accepted (`ADR-0001`).
- **Login**: manual only, in a visible browser; no credentials are ever stored, only the resulting session (`ADR-0003`).
- **Session persistence**: one stored session file per platform, reused across runs until it expires; `login <platform>` re-captures it.
- **`PlatformClient` interface** (the seam) — implementations `naukriclient`, `linkedinclient`:
  - `Login() (SessionState, error)`
  - `Search(filters Filters) ([]Profile, error)`
  - `DownloadResume(profileID string) ([]byte, error)`
- **`LLMProvider` interface** (the seam) — implementation at launch: Claude API client:
  - `ExtractFilters(jd string) (Filters, error)`
  - `Rank(jd string, profiles []Profile) ([]Match, error)`
- **Orchestrator/pipeline package** wiring both seams together: extract filters → CLI confirm/edit prompt → search both platforms (capped top 50 each) → dedup/merge into Candidates → rank via `LLMProvider` → keep top 20-50 by Match Score → download resumes for top N (default 10) → write output.
- **Output writer**: one folder per Fetch Run; a JSON file of Candidates (each holding one or more Profiles plus an optional Resume file path); a `resumes/` subfolder.
- **CLI package**: subcommands `login <platform>`, `fetch --jd <file> [--source naukri|linkedin|both] [--top-n N]`, `runs`. `--source` (case-insensitive, default `both`) is resolved entirely in the CLI layer — it selects which `PlatformEntry` values get passed into the `Orchestrator`; the orchestrator/pipeline package itself stays source-count-agnostic.
- **Config**: LLM provider selection via env var/config file (e.g. `LLM_PROVIDER=claude`) plus the Claude API key via env var. No platform credentials are configured anywhere.
- **Dedup rule**: merge on exact email or phone match; fall back to fuzzy match on name + current company, flagging fuzzy merges in the output for manual review.
- **Resilience**: CAPTCHA/2FA pauses the visible browser for manual resolution, then the run continues; LLM calls retry with backoff, then persist the raw pool and stop if still failing.
- **Domain vocabulary** (`CONTEXT.md`): `JD`, `Candidate`, `Profile`, `Resume`, `Match Score`, `Fetch Run`, `Source` — used consistently in code, CLI output, and file/field naming.
- **ADRs to respect**: `0001` (automation over official APIs), `0002` (Go + go-rod), `0003` (manual login, no stored credentials).

## Testing Decisions

- Good tests here assert **observable pipeline behavior**: given fake `PlatformClient`s returning canned Profiles and a fake `LLMProvider` returning canned Filters/Matches, does the orchestrator produce the correct merged, ranked, capped Candidate list and output files — not *how* `go-rod` drives a page or *how* a Claude API request is shaped.
- **Tested via the seam**: the orchestrator/pipeline package, dedup/merge logic, top-N resume-download selection, the output writer, and CLI command wiring — all exercised with fake `PlatformClient` and `LLMProvider` implementations injected at construction.
- **Not unit-tested**: the `naukriclient`/`linkedinclient` `go-rod` automation internals and the real Claude API `LLMProvider` implementation — inherently network/browser/ToS-sensitive, verified manually against the operator's real accounts instead.
- No prior art exists in this repo (greenfield). First tests establish the pattern: standard Go table-driven tests (`_test.go` alongside each package), fakes passed in as constructor params, no external mocking framework needed given the small interface surface.

## Out of Scope

- PDF/DOCX JD input (Markdown only for now; likely future addition).
- Any platform other than Naukri and LinkedIn.
- Official/partner API integration (rejected in `ADR-0001`).
- Automated login or credential storage (rejected in `ADR-0003`).
- Multi-user/multi-tenant support, hosting as a service, or a web UI — this is a personal local CLI tool.
- Wiring up OpenAI-compatible or local-LLM providers — the `LLMProvider` interface is built for this, but only Claude API is implemented at launch.
- Automated CAPTCHA-solving or bypassing anti-bot measures beyond the `go-rod`/stealth patches — the human resolves these manually.
- Notifications, scheduling, or automatic re-runs — fetches are user-initiated, one JD at a time.
- Analytics/reporting across multiple Fetch Runs — each run is self-contained; `runs` only lists them.

## Further Notes

- This tool automates the operator's own paid recruiter accounts and knowingly accepts ToS risk (`ADR-0001`); that's a personal-use decision already made, not something for `/implement` to revisit.
- If detection/CAPTCHA frequency turns out worse than expected with `go-rod`/stealth, `ADR-0002` documents Node + Playwright as the fallback path.
- The exact native-search filter fields available on Naukri Resdex vs. LinkedIn Recruiter weren't enumerated during grilling — confirm the real field set on each platform's search UI before finalizing the `Filters` type, since the two platforms' filter vocabularies won't be identical.
