# Tasks

## 1. Account and Onboarding
- [x] Persist account-scoped onboarding completed/skipped state and route consistently after login/launch.
- [x] Add current-device logout UI.
- [x] Add password change with security-version token invalidation.
- [x] Add admin 30-minute reset-code generation and user redemption.
- [x] Reject stale tokens and inactive/deleted accounts.

## 2. Plans and Tasks
- [x] Accept and validate explicit per-date task drafts during manual plan creation.
- [x] Build generated task-draft editing with bulk fill in the create form.
- [x] Add plan detail with plan edit, invitation, lifecycle actions, and task CRUD.
- [x] Make cards navigate to detail, expose invitation, and remove edit-layout UI.
- [x] Return and display the next task when today has no scheduled task.
- [x] Add regression coverage for weekend creation and task generation.

## 3. Study Groups
- [x] Allow active members to invite and leaders to remove members.
- [x] Add confirmations and visible error/success states.
- [x] End expired groups and calculate true current-week leaderboard metrics.
- [x] Harden one-active-group transitions against races.

## 4. Subscription Messages
- [x] Publish enabled template metadata and collect real mini-program authorization results.
- [x] Hide unsupported subscription controls on H5.
- [x] Add template mapping/page configuration and admin validation/test feedback.
- [x] Add scheduled idempotent delivery with token caching and outcome logs.
- [x] Route group nudges through the real delivery pipeline.

## 5. Verification
- [x] Run backend tests, frontend/admin type checks, H5 and mini-program builds.
- [x] Strictly validate this OpenSpec change.
- [ ] Verify WeChat authorization and delivery on a real approved template.
