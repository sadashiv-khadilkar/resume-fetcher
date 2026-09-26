# 05: Naukri resume download

**What to build:** A real `naukriclient.DownloadResume` that downloads a resume file via Resdex for a given candidate, wired into the pipeline's top-N download step so only the top-ranked Naukri candidates consume Resdex credits.

**Blocked by:** 04

**Status:** ready-for-human

- [x] `naukriclient.DownloadResume(profileID)` downloads the candidate's resume file via Resdex
- [x] Only the top N ranked Naukri candidates (per ticket 01's top-N selection) trigger a download; lower-ranked candidates never consume a Resdex credit
- [x] Downloaded resume files are saved into the Fetch Run's `resumes/` subfolder and linked from that Candidate's entry in the output JSON
- [x] If a CAPTCHA/2FA/rate-limit is hit during download, the same pause-and-resume behavior from ticket 04 applies
- [ ] Verified manually against a real Naukri Resdex account, consistent with the seam's testing approach

## Implementation notes

- The `browser` interface (`naukriclient.go`) gained `DownloadResume(state, profileID) (domain.ResumeFile, error)`; `Client.DownloadResume` shares the same `ensureSession()` reuse-vs-manual-login decision as `Login`/`Search`, then delegates - same seam/testing pattern as tickets 03/04, exercised against a fake in `naukriclient_internal_test.go`.
- Top-N selection and the `resumes/` subfolder + `ResumePath` linking were already implemented in ticket 01 (`pipeline.go`'s `Run` loop, `output.WriteRun`) and needed no changes; this ticket only had to make `DownloadResume` itself real.
- The real automation (`rod.go`) navigates to `profileID` (the profile URL captured by `Search`'s scrape), pauses via the same `waitForChallengeResolution` helper `Search` uses (before and after clicking the download control, in case the click itself triggers a credit-confirmation/CAPTCHA prompt), then uses go-rod's `Browser.WaitDownload` to capture the downloaded file. A missing download control (no résumé on file, or Resdex credits exhausted) returns `ResumeFile{Available: false}` rather than an error, per this ticket's checklist item and spec item 20 ("a missing file isn't mistaken for a bug") - the pipeline already turns that into a `ResumeNote` on the Candidate.
- The actual file wait is bounded independently of go-rod's own context-timeout semantics: `wait()` runs in a goroutine and is raced against a `time.After(resumeDownloadTimeout)`, so a browser-context timeout edge case can't hang `fetch` indefinitely.
- **The `downloadResumeButtonSelector` CSS selector is unverified against a real Resdex profile page** - no browser/network access in this sandbox, same caveat as tickets 03/04's `rod.go`. Confirm and adjust it against the real DOM before relying on this in production; that's this ticket's own remaining manual-checklist item.
