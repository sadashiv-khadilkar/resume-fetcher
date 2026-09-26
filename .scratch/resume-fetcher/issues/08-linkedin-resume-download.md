# 08: LinkedIn resume/profile-PDF download

**What to build:** A real `linkedinclient.DownloadResume` that attempts a per-profile PDF export for the top-N ranked LinkedIn candidates, explicitly reporting when export isn't available for a given profile rather than silently omitting it.

**Blocked by:** 07

**Status:** ready-for-human

- [x] `linkedinclient.DownloadResume(profileID)` attempts a profile-PDF export for the given candidate
- [x] Only the top N ranked LinkedIn candidates trigger an attempt, consistent with ticket 05's approach for Naukri
- [x] Successfully exported files are saved into the Fetch Run's `resumes/` subfolder and linked from that Candidate's entry in the output JSON
- [x] When export isn't available for a profile (e.g. the candidate's settings block it), the Candidate's output entry clearly records that no Resume could be obtained, rather than appearing as a silent omission or a bug
- [x] If a CAPTCHA/2FA/rate-limit is hit during export, the same pause-and-resume behavior from ticket 07 applies
- [ ] Verified manually against a real LinkedIn Recruiter account, consistent with the seam's testing approach

## Implementation notes

- The `browser` interface (`linkedinclient.go`) gained `DownloadResume(state, profileID) (domain.ResumeFile, error)`; `Client.DownloadResume` shares the same `ensureSession()` reuse-vs-manual-login decision as `Login`/`Search`, then delegates - same seam/testing pattern as tickets 06/07, exercised against a fake in `linkedinclient_internal_test.go` (four new tests mirroring `naukriclient_internal_test.go`'s `TestDownloadResume_*` cases: reuse-and-delegate, no-persisted-session fallback, browser-error propagation, manual-login-failure short-circuit, and the unavailable-resume-is-not-an-error case).
- "No Resume could be obtained" needed no new field: ticket 01/05 already gave `domain.Candidate` a `ResumeNote` string alongside `ResumePath`, and `pipeline.go`'s `Run` loop already sets `c.ResumeNote = "resume not available"` whenever a `PlatformClient.DownloadResume` returns `domain.ResumeFile{Available: false}` (as opposed to returning an error). `linkedinclient`'s real `DownloadResume` reuses that exact contract: a candidate whose settings block export, or for whom Recruiter shows no export control at all, comes back as `ResumeFile{Available: false}, nil` rather than an error - the same convention ticket 05 established for Naukri's "no résumé on file" case. This keeps the output JSON's "no Resume obtained" signal identical across both platforms and required no `pipeline`/`output`/`domain` changes.
- Top-N selection and the `resumes/` subfolder + `ResumePath` linking were already implemented in ticket 01 (`pipeline.go`'s `Run` loop, `output.WriteRun`) and needed no changes; this ticket only had to make `linkedinclient.DownloadResume` itself real.
- The real automation (`rod.go`) navigates to `profileID` (the profile URL captured by `Search`'s scrape), confirms the session is still valid (`requireLoggedIn`), pauses via the same `waitForChallengeResolution` helper `Search` uses (before and after clicking the export control, in case the click itself triggers a challenge prompt), then uses go-rod's `Browser.WaitDownload` to capture the exported PDF - mirroring `naukriclient/rod.go`'s `DownloadResume` shape exactly, down to racing the file wait against an independent `time.After(resumeDownloadTimeout)` so a browser-context timeout edge case can't hang `fetch` indefinitely. A missing export control (`exportPDFButtonSelector`) returns `ResumeFile{Available: false}` rather than an error, per this ticket's checklist item.
- **The `exportPDFButtonSelector` CSS selector, along with `recruiterURL`/`recruiterSearchURL` and the other selectors carried over from ticket 07, is unverified against a real LinkedIn Recruiter profile page** - no browser/network access in this sandbox, same caveat as tickets 06/07's `rod.go`. Confirm and adjust it against the real DOM before relying on this in production; that's this ticket's own remaining manual-checklist item.
