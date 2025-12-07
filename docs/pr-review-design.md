# GitHub PR Review Agent Design

KISS version of the PR-review workflow that leans on the brain LLM for all text reasoning and uses the existing `review_code` / `codex` agents. The orchestrator only coordinates branches, enforces exit conditions.

## Goals

- Find every potential P0/P1 issue in a PR using `review_code`.
- Have at least two codex agents confirm each issue by reproducing it with a minimal failing test.
- Let the codex agents exchange reasoning/test plans until they either agree or document that they disagree.
- Keep the workflow simple and deterministic so `dev-agent` can stop cleanly.

## High-Level Flow (branch-per-issue)

```
Per branch:
 ├─ run review_code once to find the top P0/P1 issue
 ├─ run codex-alpha and codex-beta to verify
 ├─ if disagreement, one short exchange round
 └─ emit confirmed / unresolved with transcripts
```

## Branch Setup

- No special pre-flight step. Each branch inherits the parent workspace state; the agent does not run `gh review` or check out refs. All prompts contain only the PR/task text.

## Phase 1 – Single-Issue Discovery

1. Run `execute_agent` with `agent=review_code` and the PR/task text.
2. Read `<workspace>/code_review.log` from that branch.
3. Treat the log content as the single candidate issue for verification. If the log is empty, exit clean.

## Phase 2 – Confirmation (per issue)

1. Run `codex-alpha` and `codex-beta` via `execute_agent` on the same issue.
   - Prompts ask them to inspect the repo/PR diff, describe reproduction, and propose a minimal failing test.
2. Consensus check via brain:
   ```
   {"agree":true/false,"explanation":"..."}
   ```
3. If they disagree, run **one** exchange round: each agent sees the peer transcript and updates their verdict/test.
4. Final verdict: `confirmed` if agree, otherwise `unresolved`, recording both transcripts and the explanation.

## Completion & Exit Conditions

- If Phase 1 yields no issue → status `clean`, summary notes that review_code reported no blocking issues.
- Otherwise return a single issue report with status `confirmed` or `unresolved`, plus transcripts and branch IDs for traceability.

## Pseudocode Sketch

```pseudo
function run_pr_review(ctx):
    log = run_review_code(ctx.task)
    issue = pick_top_issue(log)
    if issue is none: return {status:"clean"}

    alpha = run_codex("codex-alpha", issue)
    beta = run_codex("codex-beta", issue)
    verdict = consensus_check(issue, alpha, beta)
    if !verdict.agree:
        alpha = run_codex("codex-alpha", issue, peer=beta)
        beta = run_codex("codex-beta", issue, peer=alpha)
        verdict = consensus_check(issue, alpha, beta)
    return summarize(issue, verdict, alpha, beta)
```

Each “magic” step (`pick_top_issue`, `consensus_check`) is a single `brain.complete` call that ingests raw text and outputs structured JSON, keeping the orchestrator small and deterministic.
