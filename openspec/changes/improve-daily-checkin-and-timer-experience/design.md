## Context

The current mini program stores planned task time separately from study sessions, but the check-in page does not expose a live timer model or an explicit daily check-in action. Task start/stop/complete endpoints currently map several user intentions onto a small status set, and makeup/postponement use free-text modal input. The change crosses the mini program, task/check-in handlers, persistence, plan schedule generation, and AI provider integration.

## Goals / Non-Goals

**Goals:**

- Make daily check-in an explicit, rewarding user action after all tasks are complete.
- Give every task a trustworthy live countdown, elapsed time, resumable pause flow, and clear early/achieved completion semantics.
- Replace ambiguous time text entry with bounded native controls.
- Support default and weekday-specific plan scheduling without restricting actual learning start time.
- Add concise, safe, stable daily motivation without making the check-in page slow or excessively tall.

**Non-Goals:**

- No competitive gamification or social sharing for motivational messages.
- No forced start at the planned clock time.
- No automatic task completion when a countdown reaches zero.
- No reopening of completed tasks in this change.
- No arbitrary recurring-calendar rule engine beyond weekday selection and overrides.

## Decisions

### Daily check-in is an explicit finalization step

Completing all tasks sets the plan/date to `eligible`, not checked in. The user taps one visible check-in button to finalize the day, update streak, and grant rewards. The button remains visible and disabled while tasks remain. Reward and finalization remain backend-idempotent.

### Timer state is derived from persisted sessions

The server remains authoritative for accumulated study minutes and active-session start time. The client computes the live display from persisted minutes plus wall-clock time since the active session started. It periodically refreshes and reconciles after lifecycle changes so backgrounding or device-clock drift does not permanently corrupt totals.

The target duration is derived from the task's planned start/end range. Actual start time does not shorten the target. Pause closes the active StudySession; resume creates a new one. All sessions accumulate on the same task.

### Countdown completion requires user confirmation

At zero, the client enters an achieved state, provides light haptic/visual feedback, and continues counting overtime. The primary control changes from `暂停` to `完成任务`. Completion closes any active session and marks the task complete in one backend transaction.

### Pause is resumable; ending learning completes the task

`暂停` closes the active StudySession, preserves accumulated time, and leaves the task incomplete. Starting again creates another StudySession and resumes from the accumulated duration. Before the target is reached, `结束本次学习` is a confirmed early-finish intention that closes any active session and marks the task completed. After the target is reached, the prominent completion label is `完成任务`. Both completion paths produce the same completed state transactionally, and completed tasks are not reopened by this change.

### Time entry uses native bounded controls

Makeup and postponement use bottom-sheet forms composed of native date/time pickers. Inputs preload existing task values and render a duration/target summary before submission. Existing backend validation remains authoritative for future times, invalid ranges, excessive makeup, and schedule conflicts. A corrected makeup session longer than 8 hours is rejected.

### Plan schedules have defaults plus weekday overrides

A plan stores a default start/end time, selected study weekdays or explicit study dates, and optional weekday/date overrides. Tasks are generated only for selected weekdays or dates. For each selected day, generation resolves its override first and then the default; unselected days produce no task. Generated tasks retain resolved planned times. Editing a plan does not silently rewrite existing tasks; future changes require explicit regeneration or batch update.

### Daily motivation is cached and bounded

The backend stores one message per user and Asia/Shanghai date. It first reads the cache, then asks the configured AI provider using aggregate user signals, then validates output length/content. Failure uses a moderated quote library. Text is at most 32 Chinese-display characters and two lines; source is at most 12 characters. AI-original text is labeled `今日寄语` without a fabricated author.

### Task objective and completion reflection are separate records

DailyTask retains a concise execution objective that is required for newly created manual and AI tasks. Existing tasks without an objective remain readable and display `暂未填写任务目标`; editing such a task requires the objective to be supplied. The title is never silently copied into the objective.

Completion reflection is optional user-authored text associated with the completed task. Explicit completion opens a lightweight panel that shows objective and actual duration, accepts up to 500 Chinese-display characters, and provides both `保存心得并完成` and `跳过，直接完成`. The reflection can be added or edited later without rerunning completion side effects. A separate reflection endpoint or side-effect-free task-note update prevents note editing from touching reward/check-in transactions.

The completion panel is a bottom sheet over the daily page rather than a separate route. It shows a compact completion summary, a 3-5 line reflection field, remaining character count, a primary `保存心得并完成` action, and a secondary `跳过，直接完成` action. Closing or cancelling the sheet does not complete the task.

### Daily task cards reserve attention for the active flow

The check-in page card shows title/status, a compact objective, planned time, remaining/elapsed timing, one primary timer control, and low-emphasis `结束本次学习` before the target is reached. `暂停` remains resumable and visually distinct from completion. Makeup and postponement move to a `更多` menu and task detail. Completed cards collapse to title, total duration, and completed state, with details available on tap.

Objectives use a two-line collapsed presentation with an in-card expand/collapse affordance. The currently active task defaults to expanded so the objective stays visible during study. Expansion is transient UI state and resets when the user re-enters the page; completed cards do not expand inline.

## Risks / Trade-offs

- [Client timer drifts while backgrounded] → Recompute from server timestamps on `onShow` and after each state mutation.
- [Duplicate check-in or reward requests] → Use a transaction and existing idempotent reward marker/check-in record.
- [Early finish is mis-tapped] → Keep `结束本次学习` secondary, distinguish it from resumable `暂停`, and require confirmation that the task will be completed.
- [Weekday schedules complicate plan editing] → Hide overrides behind an optional schedule editor and always show the resolved summary.
- [AI adds latency or cost] → Cache once per user/day, enforce a short timeout, and fall back immediately.
- [Motivational content affects layout] → Validate length server-side and render in a fixed two-line card.
- [Reflection submission makes completion feel heavy] → Keep it optional, offer a direct skip action, and allow later editing.
- [Legacy tasks lack concrete objectives] → Display a non-fabricated placeholder and require an objective only when those tasks are edited.
- [Task cards become too tall or button-heavy] → Limit card content hierarchy and move low-frequency actions into `更多` and detail views.

## Migration Plan

1. Add schedule and daily-motivation persistence with backward-compatible defaults.
2. Add optional objective/reflection fields for existing rows; enforce objective on new creates and edits in application validation rather than a breaking database constraint.
3. Treat existing task planned ranges as target-duration sources; tasks without valid ranges use the existing default duration.
4. Add new APIs/response fields before switching the mini program UI.
5. Deploy backend first, then publish the updated mini program.
6. Roll back the client independently if needed; new fields remain additive.

## API and Persistence Contract

- `GET /api/checkins?date=YYYY-MM-DD` adds plan/date `eligible`, `remaining_tasks`, and `completed` fields. `POST /api/checkins` with `plan_id`, `date`, and `completed: true` is the sole finalization mutation; retries return the existing result without duplicate streak or reward effects.
- Task responses add `objective`, `reflection`, `target_minutes`, `accumulated_seconds`, `active_session`, `timer_state`, `remaining_seconds`, and `overtime_seconds`. `timer_state` is one of `pending`, `running`, `paused`, `achieved`, or `completed` and is derived server-side from task/session data.
- `PUT /api/tasks/:id/start` and `/resume` create or return the single active session. `PUT /api/tasks/:id/pause` closes that session without completing the task. `PUT /api/tasks/:id/stop` represents confirmed `结束本次学习`; it accepts optional `reflection`, closes an active session, and completes the task early. `PUT /api/tasks/:id/complete` performs the same transactional completion after the target is achieved.
- `PUT /api/tasks/:id/reflection` accepts `reflection` for an owned completed task and MUST NOT mutate timing, completion, check-in, streak, or reward state.
- Task create/update and AI preview/commit payloads add required `objective` for new or edited tasks. Legacy rows keep an empty objective until edited. `reflection` is nullable and limited to 500 Chinese-display characters.
- Plan create/update and response payloads add `default_planned_start`, `default_planned_end`, `study_weekdays` as ISO weekday integers `1..7`, optional explicit `study_dates`, and schedule overrides with `planned_start` and `planned_end`. Persist overrides with a unique plan/day key. Only selected weekdays or dates generate tasks; selected days without an override use the default range.
- `GET /api/motivation/daily?date=YYYY-MM-DD` returns `date`, `text`, `source`, and `origin` (`ai` or `library`). Persist one cache row per `(user_id, date)` with validated bounded content.
- Makeup accepts `actual_date`, `actual_start`, `actual_end`, and `reason`. Postponement accepts `date`, `planned_start`, `planned_end`, `reason`, and `confirm_conflict`; invalid ranges, makeup sessions longer than 8 hours, and unconfirmed conflicts are rejected by the backend.
