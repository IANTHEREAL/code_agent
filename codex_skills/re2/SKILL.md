---
name: re2
description: "Invariant-first deep code review / re-check: enumerate invariants and assumptions with falsification paths, trigger an adversarial scenario matrix on high-risk patterns, verify user-visible contracts, and report with evidence (Severity → Confidence)."
author: wenjun huang
---

# RE2: Invariant-First Deep Review / Re-check

This is a “more constrained” version of `re`: it promotes **invariants** and **key assumptions** to first-class citizens, and introduces **high-risk pattern triggers** plus an **adversarial scenario matrix** to increase the chance of catching subtle correctness bugs (mismatches, dropped objects, duplicate naming, unstable ordering, etc.).

## Mandatory principles

- **Coverage first**: do not only read the changed lines; chase the potential blast radius (callers, config, data, security, performance, compatibility, observability, operations).
- **Evidence first**: every conclusion must be backed by evidence (code locations, call paths, test results, runtime output, specs/docs).
- **Invariants & assumptions first**: write down the invariants and key assumptions that must hold, and check whether sibling paths share the same issue pattern (fix the class, not the instance); if you decide not to change something, you must provide “N/A” evidence and the shortest validation path.

## Workflow (execute in order)

### 0) High-risk pattern triggers (any hit escalates validation)

If you see any of the following patterns, you MUST upgrade to: **adversarial scenario matrix (≥3) + user-visible contract validation + regression-test bar**.

- **Ordering / reordering**: `sort/slices.Sort*`, reordering by ID/Name, relying on map iteration order
- **ID rewrite / reuse**: in-place object mutation, overwriting old IDs, reusing temporary objects
- **rename / swap / move**: renaming, swapping positions, migrating/merging objects
- **Parallel slice index alignment**: `for i := range A { use B[i] }` while A/B may be reordered or come from different sources
- **Temporary object overwrite**: temp/hidden/tombstone objects overwritten by later steps or “treated as the old object replacement”

### 1) Define the review scope

- Confirm what you are reviewing: `git diff` / commits / PR / patch / file list.
- List the “change surface”: add/remove/refactor/behavior change/default change/dependency change/config change/data-structure change.
- (Hint) Scale model: default worst-case anchors for very large deployments/data/object counts: `N_tidb≈100+`, `Data≈TB+`, `N_obj≈10^6 (tables/partitions/indexes/objects)`, `concurrency≈tens to hundreds of goroutines per node`. For performance reasoning, assume worst-case amplification under this model; if it’s not applicable, justify the boundary with code evidence (e.g., owner-only / background-only / DDL-only).
- Write an **invariant list (≥5)**: conditions that must always hold after this change (e.g., unique names, 1:1 ID↔object mapping, state machine convergence, external output contracts, compatibility with old data).
- Write **key assumptions (≥5)**: implicit preconditions relied on (e.g., object pairing does not depend on ordering, retry/recovery can still locate old objects), and for each one give the shortest falsification path (code point / search / assertion / unit test / minimal SQL). Assumptions should cover (when relevant): call frequency, full scans, allocation and lock contention, cross-node fanout, cache size upper bounds, etc. If you believe an assumption is irrelevant, mark it `N/A` and provide evidence (file + symbol/location).
- Mark high-risk areas: concurrency, persistence/migration, state machines, compatibility, public APIs / user-visible output.

### 1.5) Assumption validation pass (required)

For every key assumption from (1), execute the shortest “read-the-code confirm/falsify” loop and record it in the Assumption Ledger:

- Use `rg` / call-chain tracing to locate the **code point that carries the assumption** (function/method/struct field/constraint checks/comment contracts/test assertions).
- Read the code and classify the assumption:
  - `Confirmed`: explicitly guaranteed by code (or asserted / covered by tests)
  - `Contradicted`: code paths/data structures/caller semantics imply the opposite
  - `Unknown`: cannot be confirmed from code yet (missing info / too much context / needs execution or external docs)
- For `Unknown`, you MUST write the **shortest next step** (one command / minimal repro / which file to open next) to confirm; do not replace with “maybe/speculation”.

### 2) Build a blast-radius map

For each change point, do at least one “outside-in” blast-radius check:

- **Entry / interfaces**: public APIs/config/schema/storage formats/error messages/user-visible output.
- **Callers / dependents**: locate references and call paths; confirm caller semantics still hold.
- **Runtime behavior**: defaults, feature flags, fallback/rollback, retry/recovery, compatibility with old data/old clients.

### 2.5) Same-pattern usage scan (required for fixes/optimizations)

When the PR is a “fix/optimization” (not a “pure refactor/format-only” change), you MUST do a same-pattern consistency scan to avoid fixing only one spot:

- Abstract the **problem pattern** and **trigger condition** (falsifiable description), and the **solution strategy** used by this change.
- Use `rg` / call-chain tracing to find all “same-pattern usages / sibling paths” (same API/resource/scheduling pattern/keyword/config, etc.).
- For each hit, conclude: `needs fix too` / `clearly does not (reason + evidence)` / `Unknown (shortest validation step)`; do not write only “might be similar”.

### 3) Deep validation (confirm by category)

For each category, provide a “confirmed / N/A” conclusion with evidence:

- **Correctness**: boundary conditions, error branches, resource release, idempotency, state machine/convergence.
- **Invariant review**: check each invariant from (1) with evidence; focus on mismatch risks caused by ordering/ID rewrites/index alignment/object overwrite.
- **Security**: input validation, injection/traversal/deserialization, authn/authz/multi-tenancy isolation, sensitive data exposure.
- **Data & consistency**: transaction boundaries, concurrency races, migration/backfill, old-data compatibility, failure retry semantics.
- **Performance**: for each change point, write a cost model: `frequency (per-row/per-chunk/per-stmt/per-goroutine) × per-call cost (alloc/lock/RPC/scan/IO) × scale-model amplification`. Confirm frequency/amplification via call-chain evidence when possible; otherwise mark as `Unknown` and give the shortest validation path.
- **Observability**: logs/metrics/traces, diagnosability, debug toggles.
- **User-visible contracts**: `SHOW` / `information_schema` / return codes/messages and other externally observable behavior; run a minimal case through the key path at least once.
- **Tests & docs**: whether key-path tests must be added; whether behavior changes require doc updates.

### 4) Second confirmation (required)

Triggers (any hit requires this step):

- `Confidence` is `Medium` or `Low`
- OR `Severity` is `Blocker` or `High`

Requirements:

- Perform at least one “different method” verification and re-derive the reasoning: broaden reading scope, run a minimal test, compare history (`git blame` / commits).
- For `Blocker/High`: provide at least two complementary evidence types (failed repro / code path / runtime evidence).
- **Adversarial scenario matrix (≥3)**: e.g., multi-action single statement, shared resources (e.g., multi-column composite indexes), object overwrite/reuse; explicitly designed to break key assumptions.

If still not confirmable, state the missing info and the shortest validation step (one command or minimal repro).

## Output format (must follow)

### Conclusion

- Use 1–3 bullets to summarize overall risk and merge readiness.

### Key assumptions & validation (Assumption Ledger, required)

List at least the key assumptions from (1) (≥5). Each must include:

- `A#`: assumption id
- `Assumption`: a falsifiable statement
- `Status`: `Confirmed | Contradicted | Unknown`
- `Evidence`: code evidence (file + symbol/location; add call paths/assertions/tests if needed)
- `Next`: if `Unknown`, the shortest next validation step (single command or minimal reading path)

Template (example):

```text
A1. Assumption: ...
    Status: Confirmed
    Evidence: path/to/file.go:FuncName (...)
    Next: N/A
```

### Confirmed items (Checklist)

- For “correctness / invariant review / security / data / performance / compatibility / observability / user-visible contracts / tests / docs”, list evidence or mark `N/A`.

### Findings (sorted by severity → confidence)

If there are no confirmed issues but you have doubts / risk assumptions, you must still list them here (Confidence=Low) and include Verify steps.

For each finding, use these fields:

- `Severity`: `Blocker | High | Medium | Low`
- `Confidence`: `High | Medium | Low` (for `Medium/Low`, state what second-check you did; for `Blocker/High`, show second confirmation and complementary evidence)
- `What`: problem statement (reproducible / locatable)
- `Assumptions`: which key assumptions this depends on (reference `A#`), including their `Status`, and how you confirmed/falsified them by reading code. If it relies on `Unknown` assumptions, you usually cannot claim `High` confidence.
- `Evidence`: code/commands/output/spec references
- `Impact`: blast radius and worst-case consequence
- `Fix`: suggested fix (minimal change preferred)
- `Verify`: how to verify the fix (tests/commands/scenarios)

### Regression test bar (recommended but strict)

- If the change impacts “metadata / state machine / compatibility / user-visible contracts”, the conclusion MUST explicitly say whether a minimal regression test is needed (done / not done with reasons and risk).

### Appendix: Domain invariant library (optional)

- Metadata: unique names (case-insensitive), 1:1 ID↔object mapping, object pairing not dependent on slice order, no leftovers after deletion/cleanup.
- State machines: converges eventually, recoverable, no illegal intermediate states exposed after rollback.
- User-visible: `SHOW` / `information_schema` output is stable and does not leak internal prefixes/intermediate objects.
