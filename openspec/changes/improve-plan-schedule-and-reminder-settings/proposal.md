## Why

Plan schedule templates can already be updated by the backend, but plan detail only edits basic text fields. Users cannot adjust default times, study weekdays, or weekday-specific times after creation, and a schedule update can therefore affect unfinished tasks without the client explaining the impact. Reminder settings also request every enabled WeChat template at once and do not show what each reminder does or whether the backend currently has an authorization record.

## What Changes

- Add plan-detail editing for default planned time, selected study weekdays, and optional weekday time overrides.
- Preserve existing date-specific schedule overrides while editing weekday templates.
- Validate time ranges in the client and explain the backend's completed, pending, and running task behavior before applying a changed schedule.
- Redesign only the plan-detail operation area with one clear lifecycle action, subordinate maintenance actions, and destructive delete inside a simple more-actions sheet.
- Show every enabled reminder template with a frontend-owned name, purpose, current authorization record, and one-at-a-time mini-program authorization action.
- Refresh authorization records after each request and explain WeChat's one-time consumption and re-authorization behavior.
- Retain cancel-all and the H5 unsupported explanation.
- Do not restore configurable or draggable plan actions and do not change backend contracts.

## Capabilities

### Modified Capabilities

- `plan-management`: Expose safe plan schedule-template editing and a clearer fixed plan-detail action hierarchy.
- `notification`: Make enabled WeChat reminder authorization understandable, individually actionable, and state-aware.

## Non-goals

- Changing plan-card action configuration or restoring drag-to-configure actions.
- Editing date-specific overrides in this slice; existing date overrides are preserved verbatim.
- Changing reminder delivery, subscription persistence, template administration, or H5 platform support.
