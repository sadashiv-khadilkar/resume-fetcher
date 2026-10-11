# 10: Credit Checkpoint and targeted Resume download

**What to build:**
- During Screening, the agent can call `propose_resume_download(candidate, reason)` when must-haves are unknown and a Resume could settle them. The call only queues the proposal.
- After the first Screening pass, the Recruiter is shown the queue with the reason for each Candidate and the estimated credit cost, and approves all, some or none (Checkpoint 2).
- Approved downloads run, within Budget, through the existing `DownloadResume`. Those Candidates are then screened again with their Resume text extracted and included.
- The fixed "download the top 10" behavior is removed.

**Blocked by:** 04, 09

**Status:** ready-for-agent

- [ ] No download ever happens without approval. Tested: an agent that proposes downloads with no Recruiter approval leads to zero `DownloadResume` calls
- [ ] Partial approval is supported, and a download stops at the Budget limit
- [ ] Re-screening uses the Resume text (PDF/DOC text extraction) and replaces the earlier Screening Result
- [ ] An unavailable Resume is noted on the Candidate, and their first Screening Result is kept
