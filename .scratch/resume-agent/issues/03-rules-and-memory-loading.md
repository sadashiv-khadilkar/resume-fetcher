# 03: Load Team Rules, Personal Rules and Memory

**What to build:**
- At the start of each Fetch Run, `git pull` the Team Rules repo (path and remote from `.env`). If the pull fails, warn and use the local copy.
- Load, in order: `team-rules.md`, then the Personal Rules file, then the Memory file. Inject them into the agent's system context for every agentic stage.
- Load `team-config.yaml` (Budget defaults and maximums, skip window) into typed config.
- Record the Team Rules commit hash and a dirty flag in `run.json`.

**Blocked by:** 02

**Status:** ready-for-agent

- [ ] Rules and Memory loaded in the defined order. Missing Personal Rules or Memory files are fine
- [ ] Offline or failed pull: a warning, then the run continues with the local checkout. A missing Team Rules repo is an error
- [ ] `team-config.yaml` parsed and validated, with sensible built-in defaults when keys are absent
- [ ] The commit hash appears in `run.json`
- [ ] An example `team-rules.md` / `team-config.yaml` committed under `docs/examples/`, including failure-recovery guidelines for the agentic stages
- [ ] Tests use a temporary git repo fixture
