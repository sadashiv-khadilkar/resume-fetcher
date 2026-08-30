# 03: Naukri login & session persistence

**What to build:** A `login naukri` command that opens a visible, non-headless browser at Naukri's login page, waits for the operator to log in by hand, then captures and persists the resulting session so later runs can reuse it without logging in again.

**Blocked by:** 01

**Status:** done

- [x] `login naukri` launches a visible browser at the Naukri login page and does not submit any credentials itself
- [x] Once the operator completes login (including any 2FA/CAPTCHA) in that browser, the session is captured and persisted to disk
- [x] A subsequent command that needs a Naukri session reuses the persisted one without prompting for login again, as long as it's still valid
- [x] If the persisted session has expired or is otherwise invalid, the tool reports this clearly and directs the operator to run `login naukri` again
- [x] No Naukri username or password is ever written to disk or config at any point

## Implementation notes

- `internal/session`: source-agnostic, file-based persistence for `platform.SessionState` (one JSON file per `domain.Source` under a `sessions/` dir, gitignored). `Store.Load` returns `session.ErrNotFound` when nothing's been captured yet.
- `internal/platform/naukriclient`: real `platform.Client` for Naukri, using `go-rod` + `go-rod/stealth` (ADR-0002). `Login()` owns the reuse decision itself: load the persisted session, validate it, and only fall back to a fresh visible-browser manual login (reporting why) when it's missing or invalid - `login naukri` run a second time is exactly the "subsequent command" and "run login naukri again" the ticket's checklist describes, since no other command consumes a Naukri session until ticket 04.
- The go-rod automation (`rod.go`: launching a real browser, polling for manual login completion, capturing/replaying cookies) is deliberately behind a small `browser` interface so `Login`'s reuse/expiry/error-handling logic is unit-tested against a fake, per `spec.md`'s Testing Decisions - the real browser driver itself is unverified against a live Naukri account in this sandbox (no network/browser access here), same caveat as ticket 02's Claude smoke tests.
- `Search`/`DownloadResume` on `naukriclient.Client` return explicit "not implemented until ticket 04/05" errors rather than being silently unimplemented.
- `login <platform>` is a new CLI subcommand (`internal/cli/login.go`, dispatched from `main.go`); `linkedin` returns a distinct "not implemented until ticket 06" error rather than being treated as an invalid platform name.
- `/code-review` on `rod.go` (the unverified automation) flagged four issues, all fixed: `ManualLogin`/`ValidateSession` now share a `launchBrowser` helper instead of duplicating the launch/connect/cleanup sequence; `ValidateSession` is bounded by `validateSessionTimeout` so a stalled dashboard page can't hang `login naukri` forever; `waitForManualLogin` fails fast after `maxConsecutivePollErrors` consecutive `page.Info()` errors instead of silently polling the full 5-minute timeout when the operator closes the browser; and login-page detection now compares the URL's host against `loginHost` rather than a bare substring match, so a query string or callback path containing "login" can't cause a false read.
