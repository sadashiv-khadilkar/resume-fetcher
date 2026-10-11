# 13: `doctor`, README and Windows release

**What to build:**
- A `doctor` command that checks:
  - Chrome is present
  - the Naukri, LinkedIn and Zoho sessions are present and not expired
  - the LLM provider key is valid (a cheap test call)
  - the Team Rules repo is reachable and parseable

  Each check reports pass, warn or fail, with a fix hint. It warns when a local model is selected, because tool calling is weaker.
- Rename the binary and module to `resumeagent`. Update the README for the new workflow.
- Build a Windows `.exe` with a GitHub Actions release.

**Blocked by:** 02, 03, 06

**Status:** ready-for-agent

- [ ] `doctor` covers every check above, with an exit code that's non-zero on any fail
- [ ] README describes setup, the `login` commands, `fetch --job`, the Checkpoints, `verdicts.md` editing and `push`
- [ ] A release workflow produces a versioned `resumeagent.exe`
