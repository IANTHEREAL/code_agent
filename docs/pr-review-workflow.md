# GitHub PR Review Agent Design

## 1. Context & Goals

`dev_agent` already orchestrates two Pantheon agents—`codex` (builder) and `review_code` (critic)—through a strict single-call TDD loop. We now need a GitHub pull-request review flow that:

- Uses `review_code` to discover **critical (P0/P1)** issues.
- Requires an **independent codex confirmation** of each reported issue via reasoning plus a minimal failing unit test.
- After confirmation, has codex craft the **public review comment** to attach to the PR.
- Repeats until `review_code` reports *“No P0/P1 issues found.”*

The design must respect the existing tooling rules (one `execute_agent` per turn, strict branch lineage, worklog/code_review artifacts) while introducing the double-check and comment-writing stages.

## 2. High-Level Flow

ASCII swim lane (every arrow is an `execute_agent` call using the previous branch’s `branch_id` as `parent_branch_id`):

```
Start (Pantheon parent branch)
 |
 v
[review_code] ──> branch A
 |   \
 |    └─ report “No P0/P1 issues found” → Finish
 |
 v
Critical issue logged in branch A’s code_review.log
 |
 v
[codex confirm] ──> branch B
   - Skeptical RCA of the flagged code path
   - Add minimal failing unit test reproducing issue
   - Capture test output + reasoning in worklog
 |
 v
Confirmed failure? (test fails before fix)
 |
 v
[codex comment] ──> branch C
   - Draft final PR comment summarizing issue + repro steps
 |
 v
Loop back to [review_code] with parent=branch C to search for the next issue
 ```

Key differences from the default TDD workflow:

1. **Two codex branches per issue**: one to reproduce/confirm, one to produce the review comment.
2. **No “could not reproduce” branch**: by policy, confirmation must succeed before issuing a review comment. (If it cannot, orchestration aborts with FINISHED_WITH_ERROR for manual intervention.)

## 3. Detailed Steps

1. **Initial Review (review_code)**
   - Prompt emphasizes exhaustive search for P0/P1 issues and recording them in `code_review.log`.
   - If the log says “No P0/P1 issues found,” orchestration stops.
2. **Issue Confirmation (codex)**
   - Prompt includes: original task, excerpt of the critical issue from `code_review.log`, repository/workspace hints, and requirement to think skeptically.
   - Codex writes/updates the smallest possible unit test that exposes the bug (failing before fix). No fix is implemented—only reproduction plus reasoning.
   - Worklog records analysis + failing test command/output.
3. **Comment Authoring (codex)**
   - Prompt includes the confirmed issue details plus failing test output.
   - Codex writes a concise PR-ready comment (markdown) referencing evidence and test.
   - Worklog logs the comment summary and any test rerun done for context.
4. **Next Issue Loop**
   - Orchestrator invokes `review_code` again using the latest branch (with the new comment files) as parent.
   - Repeat until clean review.
5. **Publish**
   - Same as existing `dev_agent`: once clean, codex performs the final push/commit stage without touching `worklog.md` or `code_review.log`.

## 4. Prompt Considerations

- **review_code template additions**
  - Stress classification: record each critical issue with enough detail for codex to locate files and reproduce.
  - Remind reviewer not to produce fix code, only findings.

- **codex confirmation prompt**
  - Inputs: original task, excerpt of the specific review finding, workspace path, and worklog location.
  - Checklist:
    - Validate the reviewer’s claim via reasoning.
    - Add/modify a minimal test proving the issue.
    - Run the relevant test command; capture failure logs in worklog.
    - Do *not* implement fixes.

- **codex comment prompt**
  - Inputs: confirmed issue summary, failing test info, commit diff pointer if needed.
  - Output: Markdown comment ready for copy/paste into the GitHub PR, referencing test names and reproduction steps.
  - Should remind codex to store the comment text in a deterministic artifact (e.g., `artifacts/review-comments.md`) for the publisher to consume later.

## 5. Branch & Artifact Management

- **Branch Lineage**: `review_code` → `codex-confirm` → `codex-comment` → repeat. Orchestrator’s `ToolHandler` already tracks this; we just enforce it when crafting prompts.
- **Artifacts**:
  - `worklog.md`: same structure (Phase summaries, test logs, fix summaries).
  - `code_review.log`: produced only by `review_code`.
  - `review-comments.md` (new): codex comment step writes markdown entries per issue. Not staged during publish unless desired (to be decided).

## 6. Failure Handling

- If `review_code` fails to produce `code_review.log` after three attempts, workflow halts with `FINISHED_WITH_ERROR` (existing behavior).
- If codex confirmation cannot produce a failing test (e.g., reviewer false positive), codex should stop after explaining why reproduction failed. The orchestrator should treat this as fatal to keep the audit trail consistent, since the policy requires confirmation before commenting.
- Any MCP or git failures follow existing error paths (instruction propagation, stream JSON errors, etc.).

## 7. Open Questions

1. **Comment artifact placement**: Should codex write one file per issue, append to a single markdown file, or rely solely on worklog summaries?
2. **Automated PR posting**: Will a later stage read the generated comment and post it via GitHub API, or is manual copy/paste acceptable?
3. **Test naming convention**: Should we enforce a prefix like `TestRepro_<IssueId>` to make failures searchable?
4. **Maximum iterations**: Do we keep the global `maxIterations` (8 review loops) or allow more given the extra codex steps per issue?

Once these answers are locked down we can wire the prompts into `orchestrator.go` and extend the publish summary to mention generated PR comments.
