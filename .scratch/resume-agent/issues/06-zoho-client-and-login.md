# 06: Zoho Recruit client and `login zoho`

**What to build:**
- An `internal/zoho` plain-HTTP client for Recruit REST v2.
- OAuth device flow against `accounts.zoho.in` (`/oauth/v3/device/code` → poll `/oauth/v3/device/token`), using the `api_domain` returned by OAuth.
- Refresh-token storage next to the platform sessions, with automatic access-token refresh. Access tokens last 1 hour.
- `login zoho` prints the verification URL and code, then waits.
- Client methods:
  - search Candidates by email, phone and criteria
  - get Job Opening
  - get a Candidate's associated Job Openings and statuses
  - upsert Candidates (`duplicate_check_fields`)
  - upload attachment (`attachments_category=Resume`)
  - associate with a Job Opening
  - change status

**Blocked by:** None

**Status:** ready-for-human (needs a Zoho API client registered in the Zoho API console, and a test org or sandbox)

- [ ] `login zoho` completes the device flow and persists the token. Running it again refreshes the token
- [ ] The client refreshes tokens transparently and surfaces Zoho error codes clearly
- [ ] Unit tests against an `httptest` server for each method, the refresh path and error mapping
- [ ] Verified manually against a real Zoho org. Record the org's real `Candidate_Status` values for Selected/Rejected, and whether unique fields for the LinkedIn URL and Naukri ID exist (ask a Zoho admin to add them if not)
- [ ] `.env.example` documents the Zoho client ID and secret
