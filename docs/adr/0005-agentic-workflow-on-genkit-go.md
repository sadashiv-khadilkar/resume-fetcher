---
status: accepted
---

# A fixed Go workflow with agentic steps, built on Genkit Go behind our own seam

The Fetch Run moves from a fixed pipeline (the LLM is called at two hard-coded points: `ExtractFilters`, `Rank`) to a workflow whose stage order is still fixed in Go code, but where two stages are tool-calling agent loops: **search refinement** (re-run and adjust Filters until the pool is good, within Budget) and **Screening** (judge each Candidate against the JD and Rules, producing a Screening Result). Everything that spends money, writes to an external system, or must never lose data — Budget enforcement, Checkpoints, Dedup against Zoho, the Zoho push, writing every seen Candidate to `verdicts.md` — stays in plain Go code, not in the prompt. Tactical recovery inside the agentic stages (0 results, a selector timeout, a malformed response) is returned to the agent as a tool result and handled per the Rules.

A single free-roaming agent with every tool was rejected: it leaves credit spend and Zoho writes to prompt-following. Multi-agent handoffs (sourcer → screener → pusher) were rejected as handoff overhead with no benefit at this scale.

The agent runtime is **Genkit Go** (`github.com/firebase/genkit/go`, 1.x), chosen over CloudWeGo Eino (pre-1.0, no native OTel), LangChainGo (no HITL, thinning maintenance), Google ADK Go (no Claude or native Ollama in Go) and Microsoft Agent Framework Go (v0.1 preview). Genkit is the only option evaluated with first-party Go plugins for all four required providers (Anthropic, OpenAI-compatible, Google GenAI, Ollama), OpenTelemetry at its core, and tool interrupts (`ai.InterruptWith` / `RestartWith`) for Checkpoints. Staying in Go keeps the existing go-rod clients and a single-`.exe` distribution (`ADR-0002`). Switching language (Pydantic AI was the runner-up) was rejected for that reason.

Genkit is used only behind an internal `agent.Loop` seam so that a later move (e.g. to Microsoft Agent Framework Go once it reaches 1.x) is contained. Genkit has no checkpoint store; none is needed because the only long pause — Verdict review — is file-based (`verdicts.md` + `push <run>`), not a suspended agent.

This supersedes the `LLMProvider.Rank` / Match Score design from the original spec. The model is one operator-selectable model (Claude, OpenAI, Gemini, or local via Ollama), not per-step.
