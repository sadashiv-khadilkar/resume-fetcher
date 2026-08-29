# 05: Naukri resume download

**What to build:** A real `naukriclient.DownloadResume` that downloads a resume file via Resdex for a given candidate, wired into the pipeline's top-N download step so only the top-ranked Naukri candidates consume Resdex credits.

**Blocked by:** 04

**Status:** ready-for-agent

- [ ] `naukriclient.DownloadResume(profileID)` downloads the candidate's resume file via Resdex
- [ ] Only the top N ranked Naukri candidates (per ticket 01's top-N selection) trigger a download; lower-ranked candidates never consume a Resdex credit
- [ ] Downloaded resume files are saved into the Fetch Run's `resumes/` subfolder and linked from that Candidate's entry in the output JSON
- [ ] If a CAPTCHA/2FA/rate-limit is hit during download, the same pause-and-resume behavior from ticket 04 applies
- [ ] Verified manually against a real Naukri Resdex account, consistent with the seam's testing approach
