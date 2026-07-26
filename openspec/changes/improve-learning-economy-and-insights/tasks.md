# Tasks

## 1. Planning Rules

- [x] Preserve a requested first hour and exclude future tasks from profile history.
- [x] Enforce and display the fully-contained 30-minute overlap rule.
- [x] Add duration and overlap boundary tests.

## 2. Group Experience

- [x] Route invitations to the registered group page.
- [x] Load group sections independently with loading and empty states.
- [x] Add mini-program share metadata and route regression coverage.

## 3. Slack Economy

- [x] Settle sessions with exact signed deltas and allow bounded session debt.
- [x] Block new sessions while balance is non-positive and expose the reason.
- [x] Let daily check-in rewards repay debt naturally.
- [x] Add a configurable low-balance subscription reminder.
- [x] Add debt settlement regression coverage.

## 4. Statistics

- [x] Add server-owned 7-day, 1-month, and 1-year trend ranges.
- [x] Add time and plan dimensions with planned, actual, overtime, and completion metrics.
- [x] Add responsive H5/mini-program bar visualizations and states.
- [x] Add aggregate regression coverage.

## 5. Admin Operations

- [x] Add four operations charts and top-level metrics.
- [x] Define mutually exclusive active, general, zombie, banned, inactive, and deleted user segments.
- [x] Record the latest successful user login time and method.
- [x] Show OpenID and last login in user management.

## 6. Verification

- [x] Run Go tests and build.
- [x] Run frontend and admin type-checks.
- [x] Run H5, WeChat, and admin builds.
- [x] Run strict OpenSpec validation and diff checks.
