---
status: accepted
---

# Secrets and config are loaded from a gitignored `.env` file via `godotenv`

Unlike Naukri/LinkedIn login (`ADR-0003`), the tool does hold one real secret: the LLM provider's API key (`ANTHROPIC_API_KEY`, plus `ANTHROPIC_WORKSPACE_ID`/`ANTHROPIC_MODEL`/`LLM_PROVIDER`). Rather than requiring these as CLI flags or bare environment variables, `cmd/resumefetcher/main.go` loads them from a `.env` file (via `github.com/joho/godotenv`) before dispatching to a subcommand, with `.env.example` as the checked-in template and `.env` itself gitignored.

Values already present in the real environment take precedence over the file, so CI or shell-exported overrides still work without editing `.env`. This is the precedent for how any future secret/config value gets supplied to the tool, rather than each ticket inventing its own mechanism.
