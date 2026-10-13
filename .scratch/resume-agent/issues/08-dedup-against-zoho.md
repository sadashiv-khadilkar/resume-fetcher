# 08: Pre-credit Dedup against Zoho, with the skip window

**What to build:** After search and before Screening or any credit spend, look up each merged Candidate in Zoho:
- by LinkedIn URL or Naukri ID (once those fields exist, see ticket 06)
- otherwise by email or phone if visible
- otherwise by name + current company (a fuzzy match, flagged)

A Candidate is skipped if any of their Job Opening associations had a status change inside the skip window (`team-config.yaml`, default 180 days). Older matches are treated as fresh. Skipped Candidates still appear in `verdicts.md` under a "Skipped (already in Zoho)" section with a link to their Zoho record.

**Blocked by:** 05, 06

**Status:** ready-for-agent

- [ ] Lookup order as above, with fuzzy matches marked as such
- [ ] Skip-window rule tested at the boundary (179, 180 and 181 days)
- [ ] Skipped Candidates listed, never silently dropped. The skip count is in `run.json`
- [ ] Zoho API failure: warn and continue without Dedup, with the run flagged. Push-time Dedup (ticket 11) still protects against duplicates
