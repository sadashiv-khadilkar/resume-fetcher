# 08: LinkedIn resume/profile-PDF download

**What to build:** A real `linkedinclient.DownloadResume` that attempts a per-profile PDF export for the top-N ranked LinkedIn candidates, explicitly reporting when export isn't available for a given profile rather than silently omitting it.

**Blocked by:** 07

**Status:** ready-for-agent

- [ ] `linkedinclient.DownloadResume(profileID)` attempts a profile-PDF export for the given candidate
- [ ] Only the top N ranked LinkedIn candidates trigger an attempt, consistent with ticket 05's approach for Naukri
- [ ] Successfully exported files are saved into the Fetch Run's `resumes/` subfolder and linked from that Candidate's entry in the output JSON
- [ ] When export isn't available for a profile (e.g. the candidate's settings block it), the Candidate's output entry clearly records that no Resume could be obtained, rather than appearing as a silent omission or a bug
- [ ] If a CAPTCHA/2FA/rate-limit is hit during export, the same pause-and-resume behavior from ticket 07 applies
- [ ] Verified manually against a real LinkedIn Recruiter account, consistent with the seam's testing approach
