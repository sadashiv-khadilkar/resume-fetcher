# 03: Naukri login & session persistence

**What to build:** A `login naukri` command that opens a visible, non-headless browser at Naukri's login page, waits for the operator to log in by hand, then captures and persists the resulting session so later runs can reuse it without logging in again.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `login naukri` launches a visible browser at the Naukri login page and does not submit any credentials itself
- [ ] Once the operator completes login (including any 2FA/CAPTCHA) in that browser, the session is captured and persisted to disk
- [ ] A subsequent command that needs a Naukri session reuses the persisted one without prompting for login again, as long as it's still valid
- [ ] If the persisted session has expired or is otherwise invalid, the tool reports this clearly and directs the operator to run `login naukri` again
- [ ] No Naukri username or password is ever written to disk or config at any point
