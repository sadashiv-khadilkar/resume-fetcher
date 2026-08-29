---
status: accepted
---

# Browser automation against the operator's own logged-in sessions, not official APIs

Neither Naukri Resdex nor an individual LinkedIn Recruiter seat exposes a self-serve public API — LinkedIn's Talent Solutions API is enterprise/partner-gated, and Resdex is a web-UI product with no documented public API. To reach data the operator's paid seats are already entitled to see, this tool drives a real browser logged in as the operator on each site, rather than waiting on or seeking official API partnership.

This is a deliberate trade-off: both platforms' Terms of Service prohibit automated/bot access regardless of account type, so this carries real risk of the operator's account being flagged or suspended. The risk is accepted and mitigated by keeping volume low (20-50 candidates per run, see `CONTEXT.md`) and throttling requests conservatively, rather than by seeking a compliant alternative.
