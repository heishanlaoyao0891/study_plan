## Context

`PUT /api/plans/:id` already accepts `default_planned_start`, `default_planned_end`, `study_weekdays`, and `schedule_overrides`. During a schedule mutation the backend preserves completed tasks, resolves date overrides before weekday overrides and defaults, recalculates non-completed tasks, validates the complete resulting schedule, and commits atomically. Plan detail currently submits none of these fields.

Notification metadata returns only enabled, valid reminder templates as `reminder_type` and `template_id`. The subscriptions endpoint returns the user's persisted authorization records. User-friendly reminder names and purposes are stable product copy and do not require expanding the backend metadata contract.

## Decisions

### Edit the schedule in the existing plan sheet

The plan edit sheet adds default time pickers, seven weekday toggles, and an optional custom range for each selected weekday. Weekday overrides are sent together with unchanged date overrides from the loaded plan. Existing weekday overrides remain in local state when a weekday is temporarily deselected, so toggling selection does not silently discard customization.

The client rejects an empty weekday selection unless the plan already uses explicit study dates, a reversed/equal default range, and any invalid weekday override. If schedule fields differ from the loaded plan, a confirmation states how many pending and running tasks will be recalculated and that completed tasks retain their historical times. Schedule fields are omitted from title/description/target-only updates so those edits cannot accidentally invoke backend task recalculation. Backend validation remains authoritative and failure leaves the editor open.

### Use a fixed action hierarchy

Plan detail uses pause/resume as the prominent lifecycle action. Edit, invite, and more remain secondary. Delay and delete move into a focused more-actions sheet, with delete visually muted and protected by its existing detailed confirmation. Pause/resume receives a lifecycle-specific confirmation. No user action-layout API is used or restored.

### Authorize one reminder at a time

The mini-program loads enabled template metadata and persisted subscriptions together. Frontend mapping supplies each known reminder's display name and purpose. Each card invokes `requestSubscribeMessage` with exactly one template ID from the user's direct tap, submits the reminder type, template ID, and result, and reloads both metadata and subscriptions. The backend accepts the result only when both identifiers match the current enabled template, so a reused template ID or stale client cannot mutate another reminder type.

An authorization record is presented as current backend state, not as a guarantee of unlimited sends. Copy explains that a WeChat subscription authorization is generally consumed by one delivery and that users should tap authorize again when they want another reminder opportunity. Cancel-all remains available and refreshes the list after deletion. H5 renders explanation only and never calls authorization APIs.

## Risks / Trade-offs

- A persisted subscription record cannot prove whether WeChat has already consumed the grant -> label it as the saved authorization record and explain re-authorization rather than claiming an exact remaining count.
- Schedule edits can change a running task because this is existing backend behavior -> require an explicit impact confirmation that distinguishes running, pending, and completed tasks.
- Date overrides are not editable in this slice -> preserve them in every update payload and state that they continue to take priority.

## Migration Plan

No persistence migration is required. Deploy the bound subscription request contract in the backend and frontend together after strict OpenSpec validation, backend tests, frontend type-check, and H5/mini-program builds.
