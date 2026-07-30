## 1. Active session API

- [x] 1.1 Add an authenticated current-active-task endpoint that returns the existing timer view or an empty result.
- [x] 1.2 Add backend regression coverage for ownership, empty state, and elapsed duration from the original session start.

## 2. Cross-client recovery navigation

- [x] 2.1 Add a typed client API and shared post-auth routing helper that recovers an active task before the check-in route.
- [x] 2.2 Use the shared helper after H5 login and mini-program authentication while retaining onboarding routing and graceful fallback.
- [x] 2.3 Add a persistent task-list exit from task detail that preserves the active study session.

## 3. Validation and project index

- [x] 3.1 Add client regression coverage proving recovery uses the active task without start or resume mutations.
- [x] 3.2 Run backend and frontend validation, validate the OpenSpec change, and update the living project scan report.
- [x] 3.3 Add and run a regression contract for the recovered task-detail exit, then refresh validation and the project index.
