# 09: Agentic Screening and `verdicts.md`

**What to build:**
- Replace `Rank` / Match Score with Screening. Each non-skipped Candidate is screened against the JD and the Rules through `agent.Loop`, with a concurrency limit. The result is a structured Screening Result: proposed Verdict (Selected/Rejected), confidence, a must-have checklist (each must-have extracted from the JD, marked met, unmet or unknown, with quoted evidence), reasons, and 2–3 screening questions.
- After a few retries, a failure marks the Candidate unscreened.
- `fetch` writes `verdicts.md` and exits. Each section is machine-parseable: a `Verdict:` line pre-filled with the proposal, an empty `Why:` line, and the Screening Result below them.

**Blocked by:** 02, 03

**Status:** ready-for-agent

- [ ] Screening Result type plus structured-output schema. Evidence must quote the Profile or Resume text
- [ ] Every Candidate seen appears in `verdicts.md`: screened, unscreened or skipped
- [ ] `verdicts.md` round-trips: the writer and parser are tested, and hand edits to `Verdict:` / `Why:` survive parsing
- [ ] `Rank`, `MatchScore` and the related domain fields removed. The terminal summary shows each Verdict and its confidence
- [ ] Tested with the scripted fake, including the unscreened path
