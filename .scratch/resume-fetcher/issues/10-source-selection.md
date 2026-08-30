# 10: `--source` selection on `fetch`

**What to build:** A `--source` flag on `fetch` letting the operator scope a Fetch Run to Naukri only, LinkedIn only, or both, instead of always searching both platforms.

**Blocked by:** 01

**Status:** done

- [x] `fetch --source=naukri`, `--source=linkedin`, and `--source=both` each run the pipeline against only the matching `PlatformEntry`/entries
- [x] Omitting `--source` defaults to `both`, preserving today's behavior
- [x] Matching is case-insensitive (`Naukri`, `NAUKRI`, etc. all accepted)
- [x] An invalid value fails fast with an error listing the valid options (`naukri`, `linkedin`, `both`), the same way a missing `--jd` fails fast today, rather than silently defaulting
- [x] Filtering happens entirely in `internal/cli`: `Orchestrator`, `MergeProfiles`, and the rest of `internal/pipeline` are unchanged, since `Orchestrator.Platforms` already accepts any subset of entries
- [x] Covered by tests: one Candidate list per `--source` value, using the existing fake `PlatformClient`s
