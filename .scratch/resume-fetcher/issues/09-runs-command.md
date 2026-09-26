# 09: `runs` command

**What to build:** A `runs` command that lists past Fetch Run output folders so the operator can find and reopen earlier results.

**Blocked by:** 01

**Status:** done

- [x] `runs` lists past Fetch Runs, showing at least the JD used, the run's date/time, and the number of Candidates produced
- [x] Runs are listed in a sensible order (e.g. most recent first)
- [x] The command reads only from the existing per-run output folder structure established in ticket 01 — no separate run-tracking store is introduced
- [x] Covered by a test using fixture output folders, no real fetch run required

## Implementation notes

- Ticket 01's per-run output folder only ever wrote `candidates.json` and a `resumes/` subfolder - it never persisted the JD text or a start timestamp anywhere on disk, so there was nothing for `runs` to read for those two fields. Rather than add a separate run-tracking store (which the checklist explicitly rules out), `internal/output/output.go` gained a small addition to the *existing* per-run folder: `RunMeta` (`JD`, `StartedAt`) and `WriteRunMeta(runDir, meta)`, which writes `run.json` alongside `candidates.json` in the same folder `output.WriteRun` already creates. `internal/cli/fetch.go`'s `RunFetch` calls it right after `output.WriteRun` (and `writePartialOnRankFailure` calls it too, so a Rank failure still records the JD/timestamp for the raw pool it persists). This is additive metadata on the existing structure, not a new store - `runs` still discovers runs purely by scanning `--out`'s subfolders.
- `internal/cli/runs.go` implements `runs` (`RunRuns`, wired into `cmd/resumefetcher/main.go`): it scans `--out` (default `output`, matching `fetch`'s own default) for subfolders, treats a folder as a Fetch Run only if it has a `candidates.json` (skipping any stray non-run directories), and reads: candidate count from `len(candidates.json's "candidates"))`; JD and start time from `run.json` when present. Runs are printed most-recent-first as a simple table (date/time, folder name, candidate count, a one-line JD summary - the first non-blank line of the JD, Markdown heading markers stripped, truncated to 60 runes).
- **Backward-compatible fallback for pre-ticket-09 run folders**: a run folder without `run.json` (i.e. written by ticket 01-08 code before this change) is still listed rather than skipped - its date/time falls back to parsing the `YYYYMMDD-HHMMSS` prefix `output.WriteRun`'s `os.MkdirTemp` pattern gives every folder name, falling back again to the folder's mtime if that parse fails, and its JD is shown as `"(unknown - run predates run.json)"`. This judgment call keeps the "no separate run-tracking store, read only what's on disk" constraint honest for runs that predate this ticket, while still satisfying "showing at least the JD used" for every run produced going forward.
- Tests: `internal/cli/runs_test.go` (external, `cli_test` package, matching `fetch_test.go`'s convention) builds fixture run folders directly on disk - `candidates.json` plus an optional `run.json` via `output.WriteRunMeta` - with no real fetch run, LLM, or browser involved, and asserts most-recent-first ordering, correct candidate counts and JD summaries, the folder-timestamp fallback when `run.json` is absent, the "no runs found" message, and that stray non-run folders are ignored.
- `internal/cli/fetch_internal_test.go`'s `writePartialOnRankFailure` calls were updated for its new `jd`/`startedAt` parameters (needed so that path can also write `run.json`); no other tickets' test behavior changed.
