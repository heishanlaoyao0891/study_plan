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

## Existing Project Scan

Treat `docs/project-scan.html` as the living project scan and change-status index.

- Before scanning the repository for project structure, architecture, or OpenSpec status, read the report and reuse its current findings.
- After creating, updating, applying, syncing, or archiving an OpenSpec change, update the report in the same task. Keep active/archive counts, task completion, affected capabilities, requirement overlaps, dependencies, and validation status current, and append a row to the incremental update log.
- After materially changing architecture, tests, builds, deployment, or security posture, update the corresponding report sections and its last-updated date.
- Prefer incremental inspection of the changed files and affected capabilities. Perform a new full scan only when the user explicitly requests it or repository changes are too broad for reliable incremental maintenance.
