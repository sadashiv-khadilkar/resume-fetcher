# Resume Agent

A CLI agent that Recruiters run locally to find candidates on Naukri and LinkedIn for a Zoho Recruit Job Opening, using their own paid Naukri Resdex and LinkedIn Recruiter seats. It screens candidates and pushes Recruiter-confirmed Verdicts to Zoho Recruit. (Formerly "Resume Fetcher", a personal one-operator tool. See `docs/adr/0005` and `0006`.)

## Language

**Recruiter**:
A person running the tool on their own machine with their own Naukri, LinkedIn and Zoho accounts. Replaces "operator", from when the tool had a single user.
_Avoid_: User, operator

**Job Opening**:
The Zoho Recruit record a Fetch Run sources for. It is the source of the JD, and Candidates are associated with it on push.
_Avoid_: Requisition, position

**JD**:
The job description text that drives a Fetch Run: it is the input to filter extraction and to Screening. It is pulled from the Job Opening, optionally with Recruiter notes added from a Markdown file.
_Avoid_: Job posting, requirement doc

**Candidate**:
A person surfaced by a search on Naukri or LinkedIn for a given JD. A Candidate holds one Profile per Source they were found on (merged into one Candidate when the same person is found on both), optionally a Resume, and after Screening a Screening Result.
_Avoid_: Result, lead, applicant

**Source**:
Naukri or LinkedIn, a platform a Fetch Run searches. Zoho Recruit is not a Source.
_Avoid_: Platform (used for the `PlatformClient` code seam, not this Recruiter-facing concept), site

**Profile**:
Structured data about a Candidate (name, contact info, skills, experience, education) scraped from their Naukri Resdex or LinkedIn page.
_Avoid_: Resume, CV, record

**Resume**:
A downloadable file (PDF/DOC) of a Candidate's CV. Naukri downloads consume Resdex credits, so Resumes are only downloaded when approved at the credit Checkpoint.
_Avoid_: Profile, CV

**Screening**:
The agentic step that judges one Candidate against the JD and the Rules.
_Avoid_: Ranking, scoring

**Screening Result**:
The output of Screening: a proposed Verdict, confidence, a must-have checklist (each item met, unmet or unknown, with quoted evidence), reasons, and screening questions. A Candidate whose Screening failed is marked *unscreened*. Supersedes the old Match Score.
_Avoid_: Match Score, rank

**Verdict**:
Selected or Rejected for interview. The agent *proposes* a Verdict; only a Recruiter-*confirmed* Verdict is ever pushed to Zoho.
_Avoid_: Decision, status (Zoho's own field name)

**Override**:
A Recruiter changing the agent's proposed Verdict, optionally with a "why". Overrides feed Memory and the override-rate metric.

**Checkpoint**:
A point where the Fetch Run pauses for the Recruiter. There are three: Filter review, the credit Checkpoint (before Resume downloads) and Verdict review (`verdicts.md`, then `push`).
_Avoid_: Approval, gate

**Budget**:
Caps on one Fetch Run's resource use (Naukri unlocks, LinkedIn profile views, searches, LLM spend, wall time), enforced in code. Team Rules set the defaults and maximums, and a Recruiter may only lower them.

**Dedup**:
Detecting Candidates already known. *Within a run*: merging Profiles across Sources. *Against Zoho*: a pre-credit check (LinkedIn URL, Naukri ID or name + company) and a push-time check (email or phone, through upsert). A Candidate with a Job Opening status change inside the skip window (default 6 months) is skipped.

**Rules**:
Markdown instructions the agent always loads. *Team Rules* live in a shared git repo, are changed by reviewed PRs, and are pulled at the start of each run. *Personal Rules* are a local file for each Recruiter.

**Memory**:
Local judgement notes the agent writes, mainly from Overrides. Recruiters prune them. Memory never holds operational errors (those go to traces).

**Fetch Run**:
A single execution against one Job Opening, producing its own output folder (Candidates, Profiles, Resumes, `verdicts.md`, `run.json`, `trace.jsonl`). It ends at the Verdict review Checkpoint, and the push to Zoho is a separate `push <run>` step.
_Avoid_: Session, job (ambiguous with the job description itself)
