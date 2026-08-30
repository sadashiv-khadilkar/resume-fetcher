# 04: Naukri Resdex real search

**What to build:** A real `naukriclient` implementation of `PlatformClient.Search` that drives the browser (via the session from ticket 03) to run the confirmed filters as a native Resdex search and return structured Profile data. `fetch` now returns real Naukri candidates (LinkedIn side still uses the fake from ticket 01 until ticket 07).

**Blocked by:** 01, 03

**Status:** ready-for-human

- [x] `naukriclient.Search(filters)` translates the confirmed filters into Naukri Resdex's native search UI and returns structured Profile data (name, contact info, skills, experience, education) for the results
- [x] The raw result pool from this search is capped (e.g. top 50), matching the pipeline's existing cap from ticket 01
- [x] If a CAPTCHA, 2FA prompt, or apparent rate-limit is hit mid-search, the visible browser pauses and waits for the operator to resolve it, then the search continues automatically
- [x] Running `fetch --jd <file>` end-to-end now produces real Naukri candidates in the output and summary table
- [x] `naukriclient`'s browser-automation internals are not unit-tested (per the spec's seam); this ticket is verified manually against a real Naukri Resdex account

## Implementation notes

- `naukriclient.Client.Search` shares its session-reuse-vs-manual-login decision with `Login` via a new private `ensureSession()` (both previously duplicated in `Login`) - `fetch` never calls `Login` first, so `Search` has to be able to obtain a session on its own the first time it runs, exactly like `login naukri` does.
- The `browser` interface gained `Search(state, filters) ([]domain.Profile, error)`, keeping `Client.Search`'s session/delegation logic unit-tested against a fake (`naukriclient_internal_test.go`), same seam as ticket 03.
- The real automation (`rod.go`) drives `resdexSearchURL`'s native filter form (skills/experience/location/notice period) via go-rod element selectors, scrapes result cards into `domain.Profile`, and pauses the visible browser via `waitForChallengeResolution` (polling the page's rendered text for CAPTCHA/2FA/rate-limit markers) until the operator resolves it or `challengeTimeout` (5 min) elapses. `maxSearchResults` (50) defensively bounds how many cards are scraped, ahead of the pipeline's own `RawPoolCapPerPlatform` cap.
- **Every CSS selector, the search URL, and the challenge-detection text markers in `rod.go` are unverified against a real Resdex account** - no browser/network access in this sandbox, same caveat as ticket 03's `rod.go`. Confirm and adjust them against the real DOM/copy before relying on this in production; that verification is this ticket's own manual-checklist item.
- `fetch`'s `selectPlatforms` now wires `naukriclient.New(session.New(sessionDir), out)` for Naukri (reusing `login naukri`'s session store) instead of a fake; LinkedIn keeps its ticket-01 fake until ticket 07.
- Automated CLI tests (`fetch_test.go`) only exercise `--source linkedin`/`both`-via-parsing, since naukri/both now route through real browser automation that can't run in CI; `--source` parsing itself and the real-vs-fake wiring decision are covered directly by `TestParseSources` and `TestSelectPlatforms` in `fetch_internal_test.go`.
- `/code-review` on `rod.go` flagged four issues, all fixed: `profileFromCard` no longer leaves `ExternalID` empty when `candidateProfileLinkSelector` doesn't match (it falls back to an index-based `unmatched-N` id), since `pipeline.MergeProfiles`/`candidateID` key Candidates on Source+ExternalID and two empty-ID profiles would otherwise collide onto the same Candidate and silently overwrite each other's downloaded resume; `pageShowsChallenge` now also requires no result cards be present, so a candidate's own skills/bio text containing a challenge-marker substring (e.g. "OTP verification") after a successful search no longer false-positives into a 5-minute wait for a challenge that isn't there; `Search` now checks for a login-page redirect (`requireLoggedIn`) right after navigating, giving a clear "session is invalid" error instead of a confusing "element not found" if the cookies validated in ensureSession's separate browser turn out to be expired/single-use by the time Search's own browser applies them; and `Search`'s browser now carries an overall `searchTimeout` (`challengeTimeout` + 2 minutes), matching `ValidateSession`'s bounded-interaction pattern so a stalled page with no challenge marker present can't hang `fetch` indefinitely.
