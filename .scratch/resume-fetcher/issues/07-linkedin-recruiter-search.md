# 07: LinkedIn Recruiter real search

**What to build:** A real `linkedinclient` implementation of `PlatformClient.Search` that drives the browser (via the session from ticket 06) to run the confirmed filters as a native LinkedIn Recruiter search and return structured Profile data. `fetch` now returns real candidates from both platforms, exercising the dedup/merge logic from ticket 01 against genuinely overlapping real-world data for the first time.

**Blocked by:** 01, 06

**Status:** ready-for-agent

- [ ] `linkedinclient.Search(filters)` translates the confirmed filters into LinkedIn Recruiter's native search UI and returns structured Profile data for the results
- [ ] The raw result pool from this search is capped (e.g. top 50), matching the pipeline's existing cap from ticket 01
- [ ] If a CAPTCHA, 2FA prompt, or apparent rate-limit is hit mid-search, the visible browser pauses and waits for the operator to resolve it, then the search continues automatically
- [ ] Running `fetch --jd <file>` end-to-end now merges real candidates from both Naukri (ticket 04) and LinkedIn, with exact and fuzzy dedup behaving correctly against real data
- [ ] `linkedinclient`'s browser-automation internals are not unit-tested (per the spec's seam); this ticket is verified manually against a real LinkedIn Recruiter account
