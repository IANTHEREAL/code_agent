# GitHub PR Review Agent Design

KISS version of the PR-review workflow that leans on the brain LLM for all text reasoning and uses the existing `review_code` / `codex` agents. The orchestrator only coordinates branches, enforces exit conditions.

## Goals

- Find every potential P0/P1 issue in a PR using `review_code`.
- Have at least two codex agents confirm each issue by reproducing it with a minimal failing test.
- Let the codex agents exchange reasoning/test plans until they either agree or document that they disagree.
- Keep the workflow simple and deterministic so `dev-agent` can stop cleanly.

## High-Level Flow

```
Phase 1: Issue Discovery Fan-Out        Phase 2: Confirmation & Comments
 ├─ run review_code (hint #1)            For each aggregated issue:
 ├─ run review_code (hint #2)              ├─ run codex-alpha (Round 1)
 └─ run review_code (hint #3)              ├─ run codex-beta  (Round 1)
       ↓ raw code_review.log text          ├─ consensus check via brain
 Aggregate + dedupe via brain              ├─ if needed, exchange transcripts
       ↓ canonical issue list              ├─ consensus re-check
 If empty → exit clean                      └─ format PR comment / unresolved note
```

## Phase 1 – Issue Discovery Fan-Out

1. Launch three `execute_agent` calls with `agent=review_code`. Each prompt uses the same PR context, ask for P0/P1 issues using critical reasoning.
2. After each branch finishes, read `<workspace>/code_review.log` directly from that branch. The file contents stay untouched; keep them in memory alongside the producing branch ID for traceability.
3. Ask the brain LLM (one completion) to consolidate the three raw logs:
   ```
  You are aggregating P0/P1 code review reports. Here are the raw logs from three reviewers:
   ---
   [Issue A]
   ---
   [Issue B]
   ---
   [Issue C]
   ---
   Produce a concise deduplicated list of P0/P1 issues. Format:
   ```
   ISSUE 1: orignal issue statement from review_code
   ISSUE 2: orignal issue statement from review_code 
   ...
   ```

4. Exit conditions:
   - If the LLM states “No P0/P1 issues” (or emits no `ISSUE` blocks), stop and report success (“clean PR”).
   - Otherwise continue with Phase 2.

## Phase 2 – Confirmation & Discussion

For each `ISSUE n:` block:

1. **Round 1 confirmations**
   - Run `codex-alpha` and `codex-beta` via `execute_agent`. Each prompt ask to read the PR diff, and the exact issue text block.They must reason about the defect, explain how to reproduce it locally, and propose a minimal failing test. Each agent replies in free-form text; we store the branch ID and full transcript.
2. **Consensus check**
   - Ask the brain LLM to compare transcripts:
     ```
     Transcript A: ...
     Transcript B: ...
    Do these describe the same defect and failing test? Reply JSON:
    {"agree":true/false,"explanation":"..."}
     ```
   - If `agree=true`, mark the issue as confirmed and skip to comment drafting.
3. **Exchange round**
   - If `agree=false`, send each agent the other’s transcript and ask them to
     either adopt the peer’s failing test, refine their own test, or explicitly
     withdraw the issue if convinced it is invalid. This is one extra
     `execute_agent` call per agent.
   - Run the same consensus check again on the new transcripts.
4. **Verdict handling**
   - Both agree → confirmed by both agents.
   - Neither agent can agree → re-enter the exchange round. The orchestrator keeps
     looping through “exchange → consensus check” until they agree or a small
     max-attempt threshold (e.g., two exchanges) is reached. Remaining
     disagreements after the cap are labeled “unresolved” with both transcripts
     attached.

5. **Comment drafting**
   - For confirmed issues, ask the brain LLM to produce the final GitHub-ready comment:
     ```
     Write a PR review comment summarizing this issue and the failing test. 
     {original issue log}
     {confirmed transcripts}
     ```
   - The orchestrator records `{status, comment_markdown, supporting_transcripts}`
     for each issue (status ∈ {confirmed, unresolved}).

## Completion & Exit Conditions

- If Phase 1 yields no issues → exit immediately with status `clean`, note that
  three `review_code` runs found nothing.
- Otherwise, collect all `confirmed` and `unresolved` entries into the final report. Provide the Markdown comments plus references to the branch IDs/transcripts in case humans need deeper context.
- The orchestrator enforces a small maximum number of exchange rounds per issue to preserve a guaranteed exit condition.

## Pseudocode Sketch

```pseudo
function run_pr_review(ctx):
    logs = parallel_map(seed in HINTS, collect_review_log(seed, ctx))
    issues = aggregate_issues_with_brain(logs)
    if issues.empty():
        return {status:"clean", comments:[]}

    comments = []
    for issue in issues:
        transcripts = [
            run_codex("codex-alpha", issue, ctx, nil),
            run_codex("codex-beta", issue, ctx, nil),
        ]
        verdict = consensus_check(transcripts)
        if !verdict.agree:
            transcripts = [
                run_codex("codex-alpha", issue, ctx, transcripts[1]),
                run_codex("codex-beta", issue, ctx, transcripts[0]),
            ]
            verdict = consensus_check(transcripts)
        comments.append(format_comment(issue, transcripts, verdict))

    return summarize(comments)
```

Every “magic” step (`aggregate_issues_with_brain`, `consensus_check`, `format_comment`) is a single `brain.complete` call that ingests raw text and outputs either structured JSON or Markdown, keeping the orchestrator small and easy to maintain.
