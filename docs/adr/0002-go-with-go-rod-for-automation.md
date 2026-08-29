---
status: accepted
---

# Go + go-rod + go-rod/stealth for the CLI and browser automation

The CLI, LLM calls, and browser automation are all built in Go, using `go-rod` to drive the browser and its companion `go-rod/stealth` package for anti-detection. Go's stealth/anti-detection tooling is thinner than Node's `playwright-extra` ecosystem — the more battle-tested option for the automated-recruiter-scraping pattern this tool follows (see `ADR-0001`) — but `go-rod/stealth` is a real, maintained equivalent, and a single-language codebase (no bundled Node driver process, as community `playwright-go` bindings would require) was preferred over the marginal stealth-tooling maturity gap.

If detection/CAPTCHA issues turn out to be more frequent than expected in practice, revisiting Node + Playwright is the documented fallback.
