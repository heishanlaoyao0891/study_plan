# Tasks

## 1. Feedback Contract and Persistence
- [x] Define allowed categories, statuses, content/contact limits, and public response fields.
- [x] Add migration-safe response, responder, and response-time fields to feedback reports.
- [x] Add backend validation and SQLite-enforced authenticated short-window submission throttling.
- [x] Add owner-only feedback history endpoint with strict user isolation.

## 2. H5 and Mini Program Experience
- [x] Add one dedicated feedback page shared by H5 and the mini program.
- [x] Add prominent feedback navigation from account/settings and retain the settings-and-policy link.
- [x] Add category choices, bounded content, optional contact, and submit loading/error/success states.
- [x] Add the user's report history with status labels and administrator responses.
- [x] Prevent repeated taps from creating duplicate submissions.

## 3. Administrator Workflow
- [x] Add category/status filtering and report detail to the feedback inbox.
- [x] Add deliberate response set/clear and response-preserving status mutation API.
- [x] Record responder identity and response time without exposing internal metadata to users.
- [x] Add admin loading, empty, success, validation, and failure states.

## 4. Verification
- [x] Add backend tests for validation, database throttle enforcement, owner isolation, admin authorization, response preservation/clearing, and status transitions.
- [ ] Add frontend checks for entry visibility, validation, duplicate-submit prevention, history, and public responses.
- [ ] Add admin checks for filtering, response editing, status actions, and errors.
- [x] Run backend tests, frontend/admin type checks, H5 and mini-program builds.
- [x] Strictly validate and review this OpenSpec change.
