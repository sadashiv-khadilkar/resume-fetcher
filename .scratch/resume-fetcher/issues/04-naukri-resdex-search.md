# 04: Naukri Resdex real search

**What to build:** A real `naukriclient` implementation of `PlatformClient.Search` that drives the browser (via the session from ticket 03) to run the confirmed filters as a native Resdex search and return structured Profile data. `fetch` now returns real Naukri candidates (LinkedIn side still uses the fake from ticket 01 until ticket 07).

**Blocked by:** 01, 03

**Status:** ready-for-agent

- [ ] `naukriclient.Search(filters)` translates the confirmed filters into Naukri Resdex's native search UI and returns structured Profile data (name, contact info, skills, experience, education) for the results
- [ ] The raw result pool from this search is capped (e.g. top 50), matching the pipeline's existing cap from ticket 01
- [ ] If a CAPTCHA, 2FA prompt, or apparent rate-limit is hit mid-search, the visible browser pauses and waits for the operator to resolve it, then the search continues automatically
- [ ] Running `fetch --jd <file>` end-to-end now produces real Naukri candidates in the output and summary table
- [ ] `naukriclient`'s browser-automation internals are not unit-tested (per the spec's seam); this ticket is verified manually against a real Naukri Resdex account
