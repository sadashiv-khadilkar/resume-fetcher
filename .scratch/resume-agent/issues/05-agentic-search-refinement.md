# 05: Agentic search refinement

**What to build:**
- Replace the single search pass with an agentic stage. After Checkpoint 1 (Filter review, unchanged), the agent gets a `search(source, filters)` tool and a target pool size from the Rules. It searches, inspects the result counts and samples, and relaxes or tightens the Filters until the pool is good or the Budget runs out.
- Tool errors (a selector timeout, 0 results, CAPTCHA timeout) come back as tool results and are handled according to the Rules.
- A session or login failure on one Source drops that Source for the rest of the run, marks the run partial, and prints a `login <source>` hint.

**Blocked by:** 02, 03, 04

**Status:** ready-for-agent

- [ ] The search stage runs through `agent.Loop` with the `search` tool. Profiles from every search are accumulated and merged through the existing `pipeline` merge
- [ ] Each refinement (the Filters used and the result count) is recorded in `run.json`
- [ ] A Source failure leads to the partial flag and the hint, and the other Source continues
- [ ] Recruiter review of the initial Filters is unchanged
- [ ] Tested with the scripted fakes: refinement on 0 results, the Budget stopping refinement, one Source failing
