# Tasks

## 1. Spec and Scope

- [x] Create OpenSpec change for product polish work.
- [x] Define proposal scope, non-goals, and success criteria.
- [x] Add spec deltas for admin UX, mini-program visual polish, plan creation, and task/check-in completion.
- [x] Validate OpenSpec change strictly.

## 2. Admin Console Polish

- [x] Translate admin navigation, page titles, primary buttons, table headers, filters, and status labels to Chinese.
- [x] Redesign global admin styling for a warmer, more polished PC dashboard.
- [x] Improve login page wording and visual hierarchy.
- [x] Improve expired/unauthorized admin-session handling where practical.
- [x] Verify admin type-check and build.

## 3. Mini Program Visual Polish

- [x] Establish shared cute visual tokens: colors, cards, soft shadows, pill buttons, and playful accents.
- [x] Restyle the first-run/login/onboarding experience to communicate the full product value.
- [x] Restyle core tab pages: check-in, plans, slack time, stats.
- [x] Reduce dense button grids by prioritizing common actions and moving secondary actions into softer visual treatments.
- [x] Hide local-debug controls outside local/dev intent.
- [x] Verify mini-program type-check and build target.

## 4. Core Flow Fixes

- [x] Resolve manual plan creation gap so users are not left with a plan but no tasks.
- [x] Remove duplicate client-side check-in mutation after task completion.
- [x] Fix group-page conditional rendering around current group vs history.
- [x] Add user-facing error handling for high-frequency task and group actions.
- [x] Verify backend tests still pass.

## 5. Staging Login

- [x] Distinguish `staging`/`test` from strict `production` configuration validation.
- [x] Allow an explicit test build to show the mock-login entry.
- [x] Add staging environment examples for Tencent Cloud testing.
