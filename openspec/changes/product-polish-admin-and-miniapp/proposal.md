# Product Polish: Admin Console and Mini Program

## Why

The project has implemented the core study-plan, check-in, slack-time, group, AI, notification, and admin capabilities, but the current experience still feels like a functional MVP rather than a product people want to use every day.

Two issues are most visible:

- The PC admin console is mostly English, visually plain, and difficult for Chinese operators to scan quickly.
- The mini program UI is structurally usable but lacks a warm, memorable, cute product identity. Buttons, cards, empty states, and core pages feel generic.

There are also several core-flow rough edges discovered during review: manually created plans may not generate tasks, completing a task can duplicate check-in work on the client, group-page rendering depends on fragile branch structure, and debug-only entry points can appear in production-like UI.

## What Changes

- Rework the admin console information architecture into Chinese labels and operator-friendly wording.
- Restyle the admin console with a more polished PC dashboard system: warm background, clear cards, readable tables, better buttons, and consistent form controls.
- Establish a cute mini-program visual language for the main user flow, prioritizing check-in, plans, stats, slack time, login/onboarding, and task schedule.
- Fix high-friction functional issues in the core study path so UI polish does not hide broken behavior.
- Keep the change scoped to presentation and existing behavior cleanup; do not introduce a new business module.

## Non-Goals

- No new monetization or membership system.
- No new AI planning algorithm.
- No replacement of the backend framework, frontend framework, or database.
- No full redesign of every secondary page before the core paths are polished.

## Success Criteria

- A Chinese-speaking operator can use the admin console without reading English navigation or primary table labels.
- The mini program first impression is warm, cute, and distinctive while preserving existing task/check-in functionality.
- Creating a manual plan produces an understandable next step: either tasks are generated from schedule fields or the UI clearly guides users to add tasks.
- Completing a task relies on one backend-owned completion/check-in/reward flow instead of duplicate client-side check-in mutation.
- Production-facing user flows do not show local-debug controls.
