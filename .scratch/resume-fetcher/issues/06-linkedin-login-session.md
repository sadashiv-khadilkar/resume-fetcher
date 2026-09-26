# 06: LinkedIn login & session persistence

**What to build:** A `login linkedin` command mirroring ticket 03 for LinkedIn Recruiter — visible browser, manual login, session capture and reuse.

**Blocked by:** 01

**Status:** ready-for-human

- [x] `login linkedin` launches a visible browser at LinkedIn's login page and does not submit any credentials itself
- [x] Once the operator completes login (including any 2FA/CAPTCHA) in that browser, the session is captured and persisted to disk
- [x] A subsequent command that needs a LinkedIn session reuses the persisted one without prompting for login again, as long as it's still valid
- [x] If the persisted session has expired or is otherwise invalid, the tool reports this clearly and directs the operator to run `login linkedin` again
- [x] No LinkedIn username or password is ever written to disk or config at any point
- [x] The Naukri and LinkedIn sessions are managed independently — refreshing one never affects the other
- [ ] Verified manually against a real LinkedIn Recruiter account, consistent with the seam's testing approach

## Implementation notes

- `internal/platform/linkedinclient`: real `platform.Client` for LinkedIn Recruiter, mirroring ticket 03's `naukriclient` structure exactly - `Login()`/`ensureSession()` own the reuse-vs-manual-login decision (load persisted session, validate, fall back to a fresh visible-browser manual login and persist the result), behind a small `browser` seam (`ValidateSession`/`ManualLogin`) unit-tested against a fake in `linkedinclient_internal_test.go`. `Search`/`DownloadResume` on `Client` return explicit "not implemented until ticket 07/08" errors, same precedent as `naukriclient` at ticket 03.
- Session persistence needed no changes: `internal/session.Store` was already source-agnostic (one JSON file per `domain.Source` under `sessions/`), so `login linkedin` and `login naukri` write/read independent files and neither refresh affects the other.
- `internal/cli/login.go`'s `buildLoginClient` now builds a real `linkedinclient.Client` for `"linkedin"` instead of returning the ticket-06 placeholder error.
- The go-rod automation (`rod.go`) navigates to a single URL, `https://recruiter.linkedin.com/`, for both `ManualLogin`'s starting point and `ValidateSession`'s check: unauthenticated, LinkedIn redirects away from `recruiter.linkedin.com` to its login/authwall flow; authenticated, it stays there. Both `ManualLogin`'s poll loop and `ValidateSession` treat "did we land on/stay on `recruiter.linkedin.com`" as the one login-complete signal, rather than needing a separate login-page URL/host to compare against as `naukriclient` does.
- **`recruiterURL`/`recruiterHost` are unverified against the real LinkedIn Recruiter login/redirect flow** - no browser/network access in this sandbox, same caveat as tickets 03/04/05's `rod.go`. Confirm the real redirect behavior (e.g., whether an intermediate `www.linkedin.com/login` or `checkpoint` hop needs its own handling) before relying on this in production; that's this ticket's own remaining manual-checklist item.
