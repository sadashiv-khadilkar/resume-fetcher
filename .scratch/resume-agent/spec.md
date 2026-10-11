# Resume Agent

Status: ready-for-agent

Builds on `.scratch/resume-fetcher/spec.md`. Everything there still holds unless this spec overrides it. The main overrides are: ranking becomes Screening, a single operator becomes many Recruiters, and Zoho Recruit becomes the system of record.

## Problem Statement

The Resume Fetcher is a fixed pipeline with very little intelligence. It gets one shot at filters with no retry when a search comes back empty or flooded. It ranks candidates from scraped Profile data only. It spends Resdex credits on the top 10 whether or not that's useful. It never learns from the recruiter's corrections. Its output is a local folder, so recruiters still check by hand whether a candidate is already in Zoho Recruit or was contacted by a colleague, then copy candidate details into Zoho themselves.

## Solution

Recruiters run a Resume Agent locally. Given a Zoho Job Opening, it:
- loads the shared Team Rules, Personal Rules and Memory
- extracts filters (Recruiter reviews them)
- searches Naukri and LinkedIn, refining searches agentically within a Budget
- skips Candidates already in Zoho within the skip window
- screens each Candidate into a Screening Result with a proposed Verdict
- asks before spending credits on targeted Resume downloads
- writes `verdicts.md` and exits

The Recruiter reviews and Overrides verdicts at their own pace, then runs `push <run>`. That upserts Candidates into Zoho (Dedup by email or phone), attaches Resumes, associates the Candidates with the Job Opening, and sets Selected or Rejected. Overrides become Memory notes, and the override rate is recorded as a metric. People improve the Rules; the agent doesn't change them itself.

## User Stories

**Setup and auth**
1. As a Recruiter, I want to run `login zoho` once using a device code in my browser, so that the agent acts in Zoho as me, with my permissions, and stores no password.
2. As a Recruiter, I want `doctor` to check Chrome, my Naukri, LinkedIn and Zoho sessions, my LLM key and the Team Rules repo, so that I can fix setup problems before a run fails halfway.
3. As a Recruiter, I want to pick one model (Claude, OpenAI, Gemini or local Ollama) in my `.env` with my own key, so that I can use whichever provider I have access to.

**Rules and Memory**
4. As a Recruiter, I want the agent to pull the latest Team Rules at the start of every run, failing gracefully when offline, so that everyone screens by the same, current standards.
5. As a Recruiter, I want my own Personal Rules file loaded after the Team Rules, so that I can add my own preferences.
6. As a lead, I want the Team Rules commit hash recorded in each run's `run.json`, so that every Verdict can be traced to the Rules in force at the time.
7. As a Recruiter, I want the agent to write a judgement Memory note when I Override a Verdict with a reason, and I want to be able to edit or delete those notes, so that it stops repeating the same mistakes.

**Sourcing**
8. As a Recruiter, I want `fetch --job <zoho-id> [--jd notes.md]` to pull the JD from the Job Opening, so that I don't copy and paste it.
9. As a Recruiter, I still want to review and edit the extracted Filters before any search runs (Checkpoint 1).
10. As a Recruiter, I want the agent to relax or tighten the filters and search again when results are empty or flooded, within Budget and without asking me each time, so that one bad filter set doesn't ruin the run.
11. As a Recruiter, I want a broken scraper or an expired session on one Source to let the run continue on the other Source, marked partial, with a `login <source>` hint.

**Dedup against Zoho**
12. As a Recruiter, I want Candidates checked against Zoho before any credit is spent (LinkedIn URL, Naukri ID or name + company), with anyone whose Job Opening status changed inside the skip window skipped, so that I don't pay for or re-screen people already evaluated.
13. As a lead, I want the skip window to be a Team Rule (default 6 months).

**Screening**
14. As a Recruiter, I want each Candidate's Screening Result to show a proposed Verdict, confidence, a must-have checklist (met, unmet or unknown, with quoted evidence), reasons and screening questions, so that I can verify a Verdict in seconds.
15. As a Recruiter, I want Candidates screened from their Profile first, with the agent proposing Resume downloads only where must-haves are unknown, so that credits are spent deliberately.
16. As a Recruiter, I want to approve the proposed downloads with their credit cost (Checkpoint 2), after which those Candidates are screened again with their Resume.
17. As a Recruiter, I want a Candidate whose Screening fails after retries to be marked unscreened rather than dropped.

**Review and push**
18. As a Recruiter, I want `fetch` to write `verdicts.md` and exit, so that I can review whenever I like (Checkpoint 3) by editing each Verdict and adding a "why" to Overrides.
19. As a Recruiter, I want `push <run>` to upsert Selected and Rejected Candidates into Zoho (email or phone Dedup), attach any Resume, associate the Candidate with the Job Opening and set the status, with the Screening Result as comments.
20. As a Recruiter, I want `push` to be safe to re-run after a partial failure, without creating duplicates in Zoho.
21. As a Recruiter, I want the agent never to push a Verdict I haven't confirmed, and never to contact a candidate.

**Guardrails and observability**
22. As a lead, I want Budget defaults and maximums in the Team Rules config, enforced in code, which Recruiters can lower but not raise.
23. As a Recruiter, I want each run to write `trace.jsonl` (OpenTelemetry spans for LLM and tool calls) to its folder, optionally also sending them to an OTLP endpoint, so that I can zip up a failed run and send it to whoever is debugging it.
24. As a lead, I want each run's override rate and similar metrics recorded in `run.json`.

**Distribution**
25. As a Recruiter, I want to install a single Windows `.exe` from a release.

## Implementation Decisions

**Workflow** (`ADR-0005`)

The stage order is fixed in Go. Stages marked *agentic* are tool loops:
1. Load the Job Opening and JD, run `git pull` on the Team Rules, and load Rules and Memory.
2. Extract Filters, then Checkpoint 1.
3. *Agentic* search refinement.
4. Dedup against Zoho (before spending credits).
5. *Agentic* Screening for each Candidate (can run in parallel), then the credit Checkpoint, a targeted Resume download, and Screening again for those Candidates.
6. Write `verdicts.md` and `run.json`, then exit.

`push <run>` is a separate, plain-Go step: it parses `verdicts.md`, runs the push-time Dedup through upsert, attaches, associates, sets the status, writes Memory notes from Overrides, and updates metrics.

**What is code-enforced versus decided by the agent**

Code enforces:
- Budgets and Checkpoints
- Dedup against Zoho
- push idempotency, through a push log in the run folder plus upsert
- that every seen Candidate appears in `verdicts.md`
- partial-run marking

Inside the agentic stages, the agent decides how to recover from a tool error, which comes back to it as a tool result, following the Team Rules.

**Agent runtime**
- Genkit Go 1.x behind an internal `agent.Loop` seam.
- Plugins: `anthropic`, `compat_oai` (OpenAI), `googlegenai`, `ollama`.
- One model for every step, chosen by `LLM_PROVIDER` / `LLM_MODEL` in `.env` (`ADR-0004`).
- The existing `LLMProvider` seam and the `claude` package are replaced by this. `ExtractFilters` becomes a structured-output generate call.

**Tools exposed to the agent loops**
- `search(source, filters)`, wrapping `PlatformClient.Search`
- `get_profile`
- `propose_resume_download(candidate, reason)`, which only *queues* a download for the credit Checkpoint. Downloads are never executed inside the loop.

Zoho tools are not exposed to the agent. Zoho is reached only through code-driven stages.

**Zoho** (`ADR-0006`)
- A plain-HTTP REST v2 client in `internal/zoho`.
- Auth: device-flow OAuth against `accounts.zoho.in`, using the `api_domain` returned by OAuth. The refresh token is stored next to the platform sessions.
- `login zoho` subcommand.

**Rules and Memory**
- The Team Rules repo path and URL are configured in `.env`. The repo contains:
  - `team-rules.md`: guidelines, including the agent's failure-recovery guidelines
  - `team-config.yaml` (or similar): Budget defaults and maximums, and the skip window
- Personal Rules are a local file.
- Memory is a local file of dated judgement notes.
- All three are injected into the system context in that order: Team, Personal, Memory.

**Output**

The run folder gains these files:
- `verdicts.md`: one section per Candidate. Each section has a machine-parseable `Verdict:` and `Why:` line, followed by the Screening Result.
- `trace.jsonl`
- `push.log`

`run.json` gains:
- the Team Rules commit hash
- the model
- Budget usage
- the partial-run flag
- the metrics (override rate, number unscreened, number skipped by Dedup)

**CLI**

| Command | Change |
|---|---|
| `login <naukri\|linkedin\|zoho>` | adds `zoho` |
| `fetch --job <id> [--jd <file>] [--source …]` | new `--job` flag |
| `push <run>` | new |
| `runs` | gains a column showing pushed or pending |
| `doctor` | new |

`fetch --jd` alone, without `--job`, is kept for offline use, but it cannot be pushed.

**Domain vocabulary**

See `CONTEXT.md` (updated): Recruiter, Job Opening, Screening, Screening Result, Verdict, Override, Checkpoint, Budget, Dedup, Rules, Memory.

**ADRs**
- `0001`–`0004` still hold.
- `0005`: agentic workflow on Genkit Go.
- `0006`: Zoho Recruit as the system of record.

## Testing Decisions

- Keep the existing pattern: assert observable workflow behavior with fakes injected through constructors.
- New fakes:
  - an `agent.Loop` fake that replays scripted tool calls
  - a fake Zoho client
- Tested with those fakes:
  - Budget enforcement (an agent asking for more searches than the Budget allows is stopped)
  - Checkpoint ordering (no download without approval, no push of unconfirmed Verdicts)
  - Dedup against Zoho, including the skip-window boundary
  - parsing and writing `verdicts.md`, including Overrides
  - push idempotency (re-running after a simulated partial failure pushes each Candidate exactly once)
  - writing Memory notes from Overrides
  - loading Rules and recording the commit hash
- Not unit-tested, verified manually:
  - go-rod internals (as before)
  - real calls to LLM providers
  - the real Zoho API, verified against a Zoho sandbox or test org
- Ticket 01's spike verifies the assumptions about Genkit's behavior.

## Out of Scope

- Contacting candidates: no outreach, InMail or email, not even drafts.
- A central agent service. Zoho and the Team Rules git repo are the only shared state.
- The agent editing Team Rules itself, or automatic prompt or self-tuning. Improvement is done by people.
- An `eval` replay command. Metrics are collected and evaluation is manual for now.
- Choosing a model per step.
- Zoho's official MCP server.
- Reading interview outcomes back from Zoho.
- Operational errors in Memory. They go to traces.
- Platforms other than Naukri and LinkedIn as Sources. Non-Windows builds.

## Further Notes

- **Prerequisite:** the Recruiter's manual end-to-end test of the existing Naukri and LinkedIn scrapers (tickets 03–08 of `resume-fetcher` are still `ready-for-human`). Every selector is still unverified.
- **Zoho facts not yet verified:**
  - that the India API host is `recruit.zoho.in` (the code takes it from OAuth, so it shouldn't matter)
  - Recruit's webhook and notification API (not needed)
  - the exact `Candidate_Status` values in this org
  - whether the org has, or can add, unique fields for the LinkedIn URL and Naukri ID. Ask a Zoho admin, because the pre-credit Dedup is much more reliable with them.
- **Picking a local model** applies it to the Screening loop too. Expect weaker tool calling and say so in `doctor` output.
- **Storing rejected Candidates in Zoho** has been confirmed as acceptable under company policy.
