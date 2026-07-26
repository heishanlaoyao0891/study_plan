# Codex Project Instructions

Project-specific Agent Skills live under `.codex/skills`.

Before starting work, inspect the `name` and `description` frontmatter in each `.codex/skills/*/SKILL.md`. When a request matches a skill, read that skill's full `SKILL.md` and follow it for the current task. The available skills are:

- `diagnosing-bugs`: reproduce and diagnose bugs or performance regressions.
- `grilling`: stress-test a plan, design, decision, or idea one question at a time.
- `grill-me`: shorthand trigger for the `grilling` workflow.
- `improve-codebase-architecture`: scan for architectural deepening opportunities and produce a visual report.
- `prototype`: build a throwaway logic or UI prototype to answer a design question.
- `openspec-propose`: create a complete OpenSpec change proposal.
- `openspec-apply-change`: implement tasks from an OpenSpec change.
- `openspec-update-change`: revise existing OpenSpec planning artifacts without editing code.
- `openspec-explore`: investigate and clarify an idea before or during a change.
- `openspec-sync-specs`: sync change delta specs into the main specifications.
- `openspec-archive-change`: archive a completed OpenSpec change.

Treat `.codex/skills` as the canonical Codex copy. Do not edit `.opencode/skills` while performing Codex-only skill maintenance unless explicitly requested.
