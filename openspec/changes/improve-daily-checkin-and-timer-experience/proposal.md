## Why

The current check-in page exposes task operations but does not give users a clear daily check-in action or a live sense of study progress. Testing also shows that text-based time entry, irreversible early completion, and missing schedule controls make the daily learning flow easy to misuse and difficult to trust.

## What Changes

- Add a visible daily check-in action and clearly distinguish daily check-in state from individual task state.
- Add one daily motivational quote to the check-in page, generated or selected through the configured AI capability with a safe fallback when AI is unavailable.
- Show each task's planned time range, live elapsed study time, and remaining countdown where a duration is defined.
- Change task controls according to state: start, pause, resume, early finish, and achieved-state completion.
- Separate resumable pause from `结束本次学习`: pause preserves an incomplete task, while ending learning means the task is completed early.
- Replace free-text makeup and postponement input with date/time picker controls and a confirmation summary.
- Allow plan creation and editing to define daily planned start/end times while still permitting users to start or pause tasks at any time.
- Preserve accumulated study sessions across pause/resume and close any active session when the task is completed early or after reaching its target.
- Require generated daily tasks to describe a concrete learning objective instead of containing only a title and time range.
- Allow users to write an optional completion reflection as a task note when explicitly completing a task.

## Capabilities

### New Capabilities

- `daily-motivation`: Daily AI-assisted motivational content, fallback content, caching, and safe display behavior on the check-in page.
- `task-reflection`: Optional completion reflections, later review, and editing rules for completed learning tasks.

### Modified Capabilities

- `daily-checkin`: Add an explicit check-in action and clarify how plan/day check-in relates to task completion.
- `study-timer`: Add live elapsed/countdown display, state-driven session controls, explicit task completion, and picker-based makeup/postponement.
- `plan-management`: Add daily planned start/end time configuration during plan creation and editing while retaining flexible actual start times.
- `ai-plan-generator`: Require AI-generated tasks to include concise, concrete learning objectives suitable for daily execution.

## Impact

- Mini program check-in, plan, schedule, and task-detail pages.
- Backend task/check-in handlers and task state transitions.
- Plan and daily-task request/response fields for planned time ranges.
- Study session persistence and explicit completion semantics.
- AI provider usage, daily content caching, fallback content, and admin observability.
- Daily task schema, AI preview/commit payloads, task completion flow, and historical task-detail views.

## Confirmed Decisions

- Completing all tasks unlocks daily check-in eligibility but does not automatically finalize the check-in.
- The check-in button remains visible while tasks are incomplete, stays disabled, and shows how many tasks remain.
- The user explicitly taps the enabled check-in button to finalize the day, update the streak, and receive slack-time rewards.
- A task's planned start/end range defines its target duration, but does not restrict when the user may actually start.
- Starting a task begins a full target-duration countdown even when the user starts earlier or later than the planned clock time.
- Pausing stops both countdown and study-time accumulation; resuming continues from the remaining duration and preserves prior sessions.
- The primary task control follows the timer state: `开始` starts or resumes timing, and `暂停` pauses timing and changes back to `开始`.
- `暂停` closes only the active session, preserves accumulated study time, and leaves the task incomplete so the user can start it again.
- Before the target duration is reached, `结束本次学习` is the early-finish action. It is visually secondary, requires confirmation, closes any active session, and completes the task.
- After the target duration, `完成任务` becomes the prominent primary completion action. Both early finish and achieved-state completion produce the same completed task state, which is not reopened by this change.
- After the countdown reaches zero, the primary `暂停` control automatically changes into the prominent `完成任务` action.
- Reaching zero does not automatically complete or pause the task. The UI enters a `目标已达成` state, provides a light notification, and continues accumulating overtime study.
- In the achieved state, the primary control changes to `完成任务`; the task may continue recording overtime until the user confirms completion.
- Makeup and postponement use native date/time picker controls instead of free-text input.
- Makeup provides actual date, start time, end time, and reason; postponement provides target date, planned start time, planned end time, and conflict confirmation.
- Both flows preload the task's existing schedule and show a confirmation summary before submission.
- A corrected makeup session longer than 8 hours is rejected by the backend.
- Plans define a default daily planned start/end range and may optionally select study weekdays and override the time range for specific weekdays.
- Tasks are generated only for selected weekdays or explicit study dates. A selected day without an override uses the plan's default time range; unselected weekdays or dates do not generate tasks.
- Generated tasks persist their own planned start/end values. Later plan schedule edits affect future tasks only through explicit regeneration or batch update behavior.
- Planned clock times drive schedule display, reminders, and target duration but do not restrict when users may actually start or pause learning.
- Each user receives one stable motivational message per calendar day.
- The backend caches the daily message, uses the configured AI provider when available, and falls back to a moderated built-in quote library on timeout, disabled AI, or invalid content.
- AI-original text is labeled as `今日寄语` and MUST NOT invent an author or source; library quotes display their verified source.
- AI personalization may use aggregate learning signals such as streak and recent completion rate but MUST NOT expose private task content or unrelated user data.
- Daily motivation content must be concise enough to preserve the check-in page's first-screen task visibility; generation and fallback selection enforce a display-length limit instead of relying only on visual truncation.
- Motivation text is limited to 32 Chinese-display characters and two rendered lines; a verified source is limited to 12 characters. Over-limit AI output is rejected and replaced with fallback content.
- New tasks MUST include a concrete task objective. AI-generated tasks MUST include one; legacy tasks may display `暂未填写任务目标` until the user edits them, but task title MUST NOT be copied as a substitute objective.
- Explicitly completing a task opens a lightweight reflection panel. Reflection is optional, may be skipped without blocking completion, and remains editable after completion.
- Editing a completion reflection does not change task completion, study time, check-in status, streak, or rewards. Reflection text is limited to 500 Chinese-display characters.
- Daily task cards prioritize objective, live timer, and one primary timer action. Lower-frequency makeup and postponement actions move into a `更多` menu and task detail instead of competing with the primary control.
- Task objectives show up to two lines by default and expand/collapse in place. An actively timed task defaults to expanded; completed tasks remain compact and expose full objective/reflection in task detail.
- Task completion opens a compact bottom-sheet reflection panel instead of navigating away from the daily page. The panel keeps reflection optional and preserves a one-tap skip path.
