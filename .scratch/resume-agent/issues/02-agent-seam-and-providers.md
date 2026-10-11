# 02: `agent.Loop` seam on Genkit, provider selection, run traces

**What to build:**
- An internal `agent` package with a small `Loop` interface (run a tool loop over a set of tools, return structured output, surface interrupts), implemented on Genkit Go.
- Model selection from `.env` (`LLM_PROVIDER=claude|openai|gemini|ollama`, `LLM_MODEL`, provider keys).
- Every Fetch Run writes `trace.jsonl` to its folder, with optional extra export to `OTEL_EXPORTER_OTLP_ENDPOINT`.
- `ExtractFilters` re-implemented as a structured-output call through this seam. The old `llmprovider` package is retired.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `agent.Loop` interface, a Genkit implementation, and a scripted fake for tests (replays a fixed sequence of tool calls)
- [ ] Provider and model chosen from `.env`, with a clear error for unknown or missing keys. `.env.example` updated
- [ ] `trace.jsonl` written to each run folder. The run's model is recorded in `run.json`
- [ ] Filter extraction goes through the seam. `fetch` behaves as before end to end with the fake
- [ ] `internal/llmprovider` removed, with its tests ported to the new seam
