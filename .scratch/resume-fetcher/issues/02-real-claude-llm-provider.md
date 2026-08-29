# 02: Real Claude LLMProvider

**What to build:** Replace the fake `LLMProvider` with a real Claude API-backed implementation that extracts search filters from a JD and computes a Match Score ranking for a set of Profiles. Demoed against ticket 01's fake `PlatformClient`s, so this ticket verifies genuine LLM behavior without needing real scraping in place yet.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] A Claude API implementation of `LLMProvider` exists, selected via config/env var (e.g. `LLM_PROVIDER=claude`), alongside the fake from ticket 01
- [ ] `ExtractFilters` returns real, structured filters (skills, experience range, location, notice period, etc.) derived from an actual JD's text
- [ ] `Rank` returns a genuine Match Score per Profile reflecting fit against the full JD text, not just keyword overlap
- [ ] A failed or rate-limited Claude API call is retried with backoff a bounded number of times
- [ ] If retries are exhausted, the raw (unranked) candidate pool is persisted to disk and the run stops cleanly, rather than losing already-fetched data
- [ ] `ExtractFilters` and `Rank` are tested against the real API only for a smoke-level check; the bulk of ranking/extraction-consuming logic remains tested via the fake from ticket 01
