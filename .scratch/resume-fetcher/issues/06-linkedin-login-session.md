# 06: LinkedIn login & session persistence

**What to build:** A `login linkedin` command mirroring ticket 03 for LinkedIn Recruiter — visible browser, manual login, session capture and reuse.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `login linkedin` launches a visible browser at LinkedIn's login page and does not submit any credentials itself
- [ ] Once the operator completes login (including any 2FA/CAPTCHA) in that browser, the session is captured and persisted to disk
- [ ] A subsequent command that needs a LinkedIn session reuses the persisted one without prompting for login again, as long as it's still valid
- [ ] If the persisted session has expired or is otherwise invalid, the tool reports this clearly and directs the operator to run `login linkedin` again
- [ ] No LinkedIn username or password is ever written to disk or config at any point
- [ ] The Naukri and LinkedIn sessions are managed independently — refreshing one never affects the other
