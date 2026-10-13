---
status: accepted
---

# Zoho Recruit is the shared system of record, reached via its REST API with per-recruiter OAuth

The tool stops being a personal one-operator CLI: each Recruiter runs it locally on their own Naukri/LinkedIn seats, and results converge in the company's Zoho Recruit. Zoho is the *only* shared system — there is no central agent service. Team Rules are shared separately through a git repo (see `CONTEXT.md`), not through Zoho.

Integration is Zoho Recruit REST API v2 (`/Candidates/search`, `/Candidates/upsert` with `duplicate_check_fields`, `/Candidates/{id}/Attachments`, `/Candidates/actions/associate`, `/Candidates/status`), wrapped as our own Go tools. Zoho's official MCP server was rejected: its Recruit toolset has no create/upsert, attachment, or associate operations, so it could only ever be half the integration, and our own tools let code enforce invariants (e.g. a status change requires a Recruiter-confirmed Verdict). There is no official Go SDK; the client is plain HTTP.

Each Recruiter authorizes with their own Zoho account via the OAuth device flow (`accounts.zoho.in`, `/oauth/v3/device/*`), so writes carry that Recruiter's identity and permissions. The API domain is taken from the OAuth response, not hard-coded. Like `ADR-0003`, no Zoho password is stored — only the resulting refresh token, kept locally next to the platform sessions.

Both Selected and Rejected Candidates are pushed, with the Screening Result as comments, so that Dedup can skip people already evaluated (confirmed acceptable for the company's data-handling policy). The agent never writes a Verdict to Zoho on its own: only Recruiter-confirmed Verdicts are pushed (see `ADR-0005`'s Checkpoints).

The low-volume mitigation in `ADR-0001` still applies per Recruiter seat; Budgets (see `CONTEXT.md`) make it enforceable rather than conventional.
