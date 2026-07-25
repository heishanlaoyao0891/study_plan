## Why

Core user workflows are present but fragmented. Onboarding repeats across devices, users cannot log out or manage passwords, manual plans assign one objective to every date, plan details do not expose generated tasks, a valid plan with no task today looks broken, study groups have incomplete management semantics, and notification settings do not perform real WeChat authorization or scheduled delivery.

## What Changes

- Persist onboarding completion or skip per account and honor it across H5, mini program, and devices.
- Add current-device logout, authenticated password change, global token invalidation after password changes, and administrator-issued 30-minute one-time password reset codes.
- Replace the shared manual-plan objective with an editable per-date task draft list.
- Make plan cards navigate to a full plan detail page containing plan editing, invitation, lifecycle controls, and daily task create/edit/delete.
- Remove plan-card editing-layout customization and expose invitation as the sole direct card command.
- Keep check-in strictly date-scoped while showing the next scheduled task when today has none.
- Complete the study-group core lifecycle: member invitations, removal, confirmations, expiration, real weekly metrics, and safer concurrency.
- Implement mini-program-only WeChat subscription authorization, configured template metadata, scheduled idempotent delivery, group nudges, and observable delivery results. H5 explains that WeChat subscriptions are unavailable.

## Capabilities

### Modified Capabilities

- `user-auth`: Add logout, password lifecycle, token versioning, and reset codes.
- `user-onboarding`: Persist completed/skipped account state.
- `plan-management`: Add per-date task drafts, plan detail task management, next-task context, and simplified plan cards.
- `notification`: Add real authorization and scheduled delivery.
- `admin-config`: Add password-reset and complete subscription operations.

## Confirmed Decisions

- Completed and skipped onboarding are both terminal for the current onboarding version.
- Logout affects only the current device; password changes, resets, and account deactivation invalidate all prior access tokens.
- Reset codes are one-time, shown once, and expire after 30 minutes.
- Manual plan creation generates a per-study-date task list that users can bulk initialize and edit individually.
- A day without scheduled work remains empty and displays the next task rather than moving future work into today.
- Plan cards open detail; invitation is visible; edit-layout configuration is removed.
- Group QR/miniprogram-code sharing remains out of scope; invitation-code core lifecycle is completed.
- WeChat subscription messages are mini-program only; H5 does not create unusable subscriptions.
