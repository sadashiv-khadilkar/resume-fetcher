# 01: Spike — check that Genkit Go fits

Type: prototype

**What to build:** A throwaway prototype in a separate directory, not wired into the CLI. It answers the open questions about Genkit Go before we commit to `ADR-0005`:
- Does a tool interrupt pause and resume cleanly enough to implement the credit Checkpoint?
- Can OpenTelemetry spans be written to a local JSONL file?
- Can both Claude and a local Ollama model complete a tool loop with several steps?

**Blocked by:** None

**Status:** ready-for-human (needs a real Anthropic key and a local Ollama install)

- [ ] Define a fake `search` tool and a `propose_resume_download` tool. The download tool interrupts with `ai.InterruptWith`. The Go caller approves (`RestartWith`) or edits the result, then the loop resumes
- [ ] Write spans for LLM calls and tool calls to `trace.jsonl` through a file exporter on `tracing.TracerProvider()`. Check that the spans carry the model, token counts and tool name/args
- [ ] Run the same loop on `anthropic` and on `ollama` (a local model that supports tool calls). Note how reliably each one completes the loop
- [ ] Record the pinned Genkit version, and check whether `RespondWith` exists in that version (not confirmed by research)
- [ ] Write the findings under `## Answer`. If any check fails, reopen the framework decision. The fallbacks are Pydantic AI and Microsoft Agent Framework Go
