You will review an opponent's Issue List. Your default stance is: each issue may be a misread, a misunderstanding, or an edge case--unless the code evidence forces you to accept it.

Reference severity definitions (guidance, not a hard rule):
- P0 (Critical/Blocker): Reachable under default production configuration, and causes production unavailability; severe data loss/corruption; a security vulnerability; or a primary workflow is completely blocked with no practical workaround. Must be fixed immediately.
- P1 (High): Reachable in realistic production scenarios (default or commonly enabled configs), and significantly impairs core/major functionality or violates user-facing contracts relied upon (including user-visible correctness errors), or causes a severe performance regression that impacts use; a workaround may exist but is costly/risky/high-friction. Must be fixed before release.
- Lightweight evidence bar (guidance): A P0/P1 claim must be backed by clear code-causal evidence and an explicit blast-radius assessment; if it’s borderline between P1 and P2, default to P1 unless the impact is clearly narrow or edge-case only.

Goal: For each issue, run an adversarial / rebuttal-style review. Try hard to find weaknesses that would prevent it from being legitimately classified as P0/P1. If you cannot find such a weakness, be honest and acknowledge it as a real P0/P1 issue.

How to work (principles, not rigid steps):
- Evidence-first: Conclusions must come from code and build/runtime-path facts, not experience or speculation.
- Reachability matters: Confirm whether the reported behavior is reachable in default/production paths, or only under gated features, tests, unusual configs, or non-standard environments.
- Impact must be concrete: State what it actually causes (crash, data corruption, correctness break, resource leak, supply-chain/repro risk, etc.), and its scope/probability.
- Prioritize counter-evidence: Actively look for counterexamples—unreachable branches, existing guards, existing tests/coverage, runtime fallbacks, isolation boundaries, or cases where it only affects developer workflows.
- Fixes have costs: If a fix is proposed, discuss its side effects, compatibility risk, and complexity. Avoid “fixing” something in a way that creates a bigger problem.

注意

1. 解释代码，而不是猜测
2. 质疑假设
3. 面对理解空白
4. 将分析代码和 issue 看作科学实验。不要只是猜测，

Output requirements:
For each issue, give a clear verdict: P0 / P1 / P2 / Not an issue, and include the most critical supporting evidence (file path + key symbols/logic). Provide a one-sentence justification for why it does or does not deserve P0/P1 in real scenarios.

Optional strengthening (still not rigid): Any P0/P1 claim should be backed by a minimal trigger condition or a clear, code-grounded reasoning chain.

---

The Issue List:
