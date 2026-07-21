# Design: Scheduling Improvements

## Plan And Task Concepts

A plan is a long-term learning goal container, such as `Learn Go Backend`. It owns metadata such as title, description, target dates, weekly target hours, sharing/group state, and generated tasks.

A task is a concrete execution unit under a plan, scheduled for a specific date and planned time range, such as `2026-07-21 20:00-21:30 Learn Go interfaces`. Timing, completion, postponement, makeup, and check-in behavior happen at the task level.

This distinction should be visible in UI copy and API behavior:

- Plan pages manage goals and plan-level settings.
- Task views manage daily execution and schedule changes.

## Task Schedule Fields

Tasks should store planned start/end times separately from actual study sessions. This enables reminders, conflict checks, and schedule views.

## Calendar View

The frontend should show a future 7-day schedule list rather than a full calendar in MVP. Each day shows planned tasks, planned time ranges, status, study minutes, and quick actions for start, complete, postpone, and makeup.

## Midnight Boundary

The system should not automatically carry active study sessions into the next calendar day. At 23:30 the user receives the final reminder. At 00:00, any still-active session should be auto-closed at 00:00 and attributed to the previous day. The next day, the user can edit makeup start/end times if they actually studied past midnight. If the user changes the end time to 00:30, the extra 30 minutes still belongs to the previous day's task.

This keeps check-in and stats behavior consistent while allowing manual correction for real late-night study.

All date-bound scheduling behavior uses `Asia/Shanghai` as the product timezone. The backend should run scheduled jobs for 23:30 reminders and 00:00 auto-close. The mini program should also run a compensation check when opened, so missed backend jobs can be repaired.

When a session is auto-closed at midnight, the task should move into `needs_decision` state, or equivalent `pending` plus `needs_decision=true` representation, so the user can confirm completion, make up times, or postpone the task the next day.

## Postpone And Makeup

Postpone moves a task to a target date and planned time range. Makeup allows editing actual start time, actual end time, and derived study minutes.

Makeup is actual study correction and is allowed to differ from planned time. Constraints:

- Actual end must be later than actual start.
- Actual end cannot be in the future.
- A single corrected session should not exceed 8 hours without explicit rejection or warning.

Postponing to a conflicting planned time should show a warning. The user may confirm and proceed.

## Batch Operations

Future tasks can be shifted, skipped, or regenerated in scoped batches. MVP batch operation should support shifting all future tasks in a plan by a number of days, preserving planned time ranges where possible.

For MVP, batch shift applies to unfinished tasks from a user-selected start date. The default start date is tomorrow. Completed tasks are never moved.

## Completion And Check-In

Completing all tasks for a plan on a date should automatically complete that plan's check-in for the date. If a plan has only one task on a date, completing that task completes the check-in immediately. This reduces duplicate user actions.
