---
status: accepted
---

# Login is manual; no Naukri/LinkedIn credentials are ever stored

Although `ADR-0001` already accepts automation/ToS risk for search and profile scraping, the tool does not automate the login step itself. Instead it opens a visible, non-headless browser at the platform's login page and waits for the operator to log in by hand, then captures the resulting session (`go-rod`'s equivalent of storage state) for reuse across runs (see `CONTEXT.md`'s Fetch Run).

This was chosen over storing a username/password and submitting the login form automatically, specifically to avoid holding a real Naukri/LinkedIn password as a secret in this tool at all — a login only needs to happen once per session-expiry, so the convenience gain from automating it doesn't offset that liability.
