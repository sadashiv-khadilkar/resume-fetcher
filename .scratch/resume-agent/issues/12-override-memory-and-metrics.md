# 12: Memory notes from Overrides, and run metrics

**What to build:**
- During `push`, compare each confirmed Verdict with the agent's proposal. Each Override with a `Why:` becomes a dated judgement note, which the agent writes into the local Memory file. It is condensed and generalized ("services-company background acceptable for <role family> when …"), with a link back to the run.
- Overrides without a reason are counted but not memorized.
- Record these metrics in `run.json`:
  - override rate
  - Selected→Rejected and Rejected→Selected counts
  - number unscreened
  - number skipped by Dedup
  - Budget usage
- `runs` shows the override rate.

**Blocked by:** 03, 11

**Status:** ready-for-agent

- [ ] Memory notes are appended, never rewritten, and the file stays plain Markdown so Recruiters can edit it
- [ ] Only judgement notes go into Memory, never operational errors
- [ ] Metrics are computed correctly (tested with fixtures of `verdicts.md` before and after the Recruiter's edits)
