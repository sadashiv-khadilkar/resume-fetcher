# Resume Fetcher

A personal CLI tool that finds candidates on Naukri and LinkedIn matching a given job description, using the operator's own paid Naukri Resdex and LinkedIn Recruiter seats.

Given a JD, it auto-derives native search filters, runs each platform's own search, merges candidates found on both platforms, ranks the pool with an LLM against the full JD text, downloads resumes for the top matches, and writes everything to a per-run output folder plus a terminal summary.

See [`CONTEXT.md`](CONTEXT.md) for domain vocabulary and [`docs/adr/`](docs/adr) for the decisions behind the architecture.

## Requirements

- Go 1.24.5+
- A Chrome/Chromium browser (driven via `go-rod`; `login` opens a visible window for manual sign-in, `fetch` runs headless)
- Your own paid Naukri Resdex and LinkedIn Recruiter seats
- An Anthropic API key (only if using the real `claude` LLM provider — see Configuration)

## Setup

```powershell
git clone https://github.com/sadashiv-khadilkar/resume-fetcher.git
cd resume-fetcher
copy .env.example .env
```

Fill in `.env`:

```
LLM_PROVIDER=claude
ANTHROPIC_API_KEY=your-key-here
```

`LLM_PROVIDER` defaults to `fake` (no network calls, deterministic sample data) if unset. `ANTHROPIC_WORKSPACE_ID` is only required for identity-linked (SSO-issued) API keys. `ANTHROPIC_MODEL` optionally overrides the default model.

`.env` is gitignored and must never be committed.

## Usage

```powershell
go build -o resumefetcher.exe ./cmd/resumefetcher

# Log in once per platform (visible browser; session persists under ./sessions/)
.\resumefetcher.exe login naukri
.\resumefetcher.exe login linkedin

# Run a fetch against a JD
.\resumefetcher.exe fetch --jd path\to\jd.md --source both --out output

# List past runs
.\resumefetcher.exe runs --out output
```

### `login <naukri|linkedin>`

Opens a visible browser for manual login (no credentials are ever stored, only the resulting session). Re-run for a single platform to refresh an expired session without touching the other.

### `fetch`

| Flag | Default | Description |
|---|---|---|
| `--jd` | *(required)* | Path to the JD Markdown file |
| `--source` | `both` | `naukri`, `linkedin`, or `both` |
| `--out` | `output` | Base directory for Fetch Run output |

Produces a timestamped folder under `--out` containing the ranked Candidates (with Profiles and any downloaded Resumes) and a `run.json` with the run's metadata. If LLM ranking fails after retries, the raw (unranked) candidate pool is still saved.

### `runs`

Lists past Fetch Runs (most recent first) with their date/time, candidate count, and JD summary.

## Testing

```powershell
go test ./...
```

Pipeline, dedup, ranking, and CLI wiring are tested against fake `PlatformClient`/`LLMProvider` implementations. The real browser-automation clients and the live Claude API integration are verified manually, not unit-tested.
