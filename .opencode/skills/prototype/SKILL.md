---
name: prototype
description: Build a throwaway prototype to answer a design question. Use when the user wants to sanity-check whether a state model or logic feels right, or explore what a UI should look like.
---

# Prototype

A prototype is throwaway code that answers a question. The question decides the shape.

## Pick a branch

- If the question is "Does this logic / state model feel right?", build a tiny interactive terminal app that pushes the state machine through cases that are hard to reason about on paper.
- If the question is "What should this look like?", generate several radically different UI variations on a single route, switchable via a URL search param and a floating bottom bar.

## Rules

1. Mark prototype code as throwaway and locate it close to where it would be used.
2. Provide one command to run it.
3. Keep state in memory unless persistence is the specific question.
4. Skip polish, tests, and abstractions.
5. Surface state or variant clearly after every action.
6. Once a decision is validated, fold only the decision into real code and remove prototype scaffolding from main.

For UI, prefer using an existing route with `?variant=` over creating a new page. Variants must be structurally different, not just colour tweaks.
