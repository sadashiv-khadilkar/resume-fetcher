# 09: `runs` command

**What to build:** A `runs` command that lists past Fetch Run output folders so the operator can find and reopen earlier results.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `runs` lists past Fetch Runs, showing at least the JD used, the run's date/time, and the number of Candidates produced
- [ ] Runs are listed in a sensible order (e.g. most recent first)
- [ ] The command reads only from the existing per-run output folder structure established in ticket 01 — no separate run-tracking store is introduced
- [ ] Covered by a test using fixture output folders, no real fetch run required
