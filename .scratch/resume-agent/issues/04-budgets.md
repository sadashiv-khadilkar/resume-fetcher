# 04: Budgets enforced in code

**What to build:** A per-run Budget for searches, LinkedIn profile views, Naukri unlocks/downloads, LLM spend (from token usage and a price table) and wall time:
- Defaults and maximums come from `team-config.yaml`.
- Recruiters can lower values with `fetch` flags but can't raise them past the team maximum.
- Every tool wrapper checks the Budget before acting. When a limit is hit, the tool returns "budget exhausted" to the agent rather than acting, and wall time is a hard stop.

**Blocked by:** 02, 03

**Status:** ready-for-agent

- [ ] Budget type with counters, plus a check-and-consume API used by the tool wrappers
- [ ] A flag value above the team maximum is rejected with a clear message
- [ ] An exhausted Budget is returned to the agent as a tool result, and the action does not execute
- [ ] Budget usage recorded in `run.json` and printed in the run summary
- [ ] Tested with the scripted `agent.Loop` fake: an agent that asks for N+1 searches against a Budget of N gets N searches
