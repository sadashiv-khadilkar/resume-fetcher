# 07: `fetch --job` pulls the JD from a Zoho Job Opening

**What to build:**
- `fetch --job <zoho-job-opening-id> [--jd notes.md]` fetches the Job Opening, builds the JD from its description and key fields (title, experience, location, skills), and appends any Recruiter notes file.
- The Job Opening ID is recorded in `run.json` so that `push` can associate Candidates.
- `fetch --jd` alone still works for offline use, and its run is marked not pushable.

**Blocked by:** 06

**Status:** ready-for-agent

- [ ] `--job` resolves the Job Opening and fails clearly when it's not found or access is denied
- [ ] The JD text is saved into the run folder (`jd.md`) exactly as the agent saw it
- [ ] `run.json` records the Job Opening ID, or marks the run not pushable
- [ ] Tested with the fake Zoho client
