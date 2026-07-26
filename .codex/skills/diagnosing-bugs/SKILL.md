---
name: diagnosing-bugs
description: Diagnosis loop for hard bugs and performance regressions. Use when the user says "diagnose" or "debug this", or reports something broken, throwing, failing, or slow.
---

# Diagnosing Bugs

Build a tight feedback loop before hypothesising. A loop can be a failing test, curl script, CLI invocation, browser script, replayed trace, throwaway harness, property loop, bisection harness, differential loop, or a human-in-the-loop script.

## Phases

1. Build a red-capable, deterministic, fast, agent-runnable feedback loop.
2. Reproduce and minimise until every remaining element is load-bearing.
3. Generate 3-5 ranked falsifiable hypotheses before testing any.
4. Instrument one variable at a time. Tag temporary logs with a unique prefix.
5. Write a regression test at the correct seam, apply the fix, and rerun the original loop.
6. Remove temporary instrumentation and state the cause in the handoff or commit message.

No red-capable command, no Phase 2.
