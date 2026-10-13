# 11: `push <run>` — confirmed Verdicts to Zoho

**What to build:** `push <run>` parses `verdicts.md`. For each Candidate with a confirmed Verdict (Selected or Rejected; unscreened Candidates without a Recruiter Verdict are skipped and reported), it runs these steps in order:
1. upsert into Zoho (`duplicate_check_fields` email and phone, plus the LinkedIn URL and Naukri ID fields if they exist)
2. attach the Resume if there is one
3. associate the Candidate with the run's Job Opening
4. set the status using the org's mapped values, with the Screening Result summary and the Recruiter's `Why:` as comments

Each completed step is appended to `push.log`. A re-run skips steps that are already done.

**Blocked by:** 06, 07, 09

**Status:** ready-for-agent

- [ ] Refuses to push a run marked not pushable, and lists Candidates without a Verdict
- [ ] Idempotency: after a simulated failure partway through, a re-run completes with every Candidate upserted, attached, associated and status-set exactly once (tested with the fake Zoho client)
- [ ] Prints a summary: pushed, skipped, failed (with reasons)
- [ ] `runs` shows whether each run is pushed, pending or partial
- [ ] Verified manually against the Zoho test org
