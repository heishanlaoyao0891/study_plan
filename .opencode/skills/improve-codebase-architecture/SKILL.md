---
name: improve-codebase-architecture
description: Scan a codebase for deepening opportunities, present them as a visual HTML report, then grill through whichever one you pick.
---

# Improve Codebase Architecture

Surface architectural friction and propose deepening opportunities: refactors that turn shallow modules into deep ones. The aim is testability and AI-navigability.

## Process

1. Scope before scanning. Put extra weight on files that changed recently or areas named by the user.
2. Explore where understanding one concept requires bouncing between many small modules, where interfaces are nearly as complex as implementations, where tests hit shallow wrappers, and where locality is poor.
3. Present candidates as an HTML report in the OS temp directory. Each candidate should include files, problem, solution, benefits, before/after diagram, and recommendation strength.
4. Do not propose interfaces yet. Ask which candidate the user wants to explore.
5. Once a candidate is picked, run a grilling loop to walk constraints, dependencies, seam shape, and tests.

Use the vocabulary: module, interface, implementation, depth, deep, shallow, seam, adapter, leverage, locality.
