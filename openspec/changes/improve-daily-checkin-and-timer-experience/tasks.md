# Tasks

## 1. Contract and Data Model

- [x] Add delta specs for check-in finalization, task timer states, motivation, and plan schedule templates.
- [x] Decide and document API fields for check-in eligibility/finalization, task timing, session state, weekday schedules, motivation content, task objective, and reflection.
- [x] Add database fields/tables for plan schedule templates, per-user daily motivation cache, task objective, and optional completion reflection with migration-safe defaults.
- [x] Validate the change strictly before implementation.

## 2. Backend Check-in and Timer

- [x] Separate task completion from explicit plan/date check-in finalization.
- [x] Keep check-in reward and streak updates idempotent in the backend transaction.
- [x] Return target duration, accumulated minutes, active session information, overtime state, and timer-related task status.
- [x] Implement pause/resume as separate study sessions without completing the task.
- [x] Keep pause resumable while making confirmed `结束本次学习` and achieved-state completion produce the same completed task state.
- [x] Keep countdown zero from auto-completing; preserve overtime study minutes.
- [x] Add date/time validation endpoints or request fields for makeup and postponement.

## 3. Plan Scheduling

- [x] Add plan default start/end time and selected weekday schedule fields.
- [x] Add optional weekday time overrides with default fallback resolution.
- [x] Generate daily tasks with resolved planned times and preserve existing task times on template edits.
- [x] Update AI plan commit and manual plan flows to provide resolved planned times.
- [x] Require concise concrete objectives in manual task creation and AI plan schema validation.

## 4. Daily Motivation

- [x] Add daily motivation response and per-user/date cache.
- [x] Add concise AI prompt, content validation, source labeling, and moderated fallback library.
- [x] Enforce 32-character message, 12-character source, and two-line display constraints.
- [x] Ensure prompts contain only aggregate learning signals.

## 5. Mini Program Experience

- [x] Add explicit `完成今日打卡` action with disabled remaining-task state and completed state.
- [x] Add stable daily motivation card above today's tasks without pushing core tasks below the intended first screen.
- [x] Add live elapsed/remaining/overtime timer display and lifecycle reconciliation.
- [x] Replace free-text makeup/postpone dialogs with date/time picker forms and confirmation summaries.
- [ ] Configure plan default schedule, weekday selection, and weekday override UI.
- [x] Distinguish resumable pause from confirmed `结束本次学习` and highlight `完成任务` after the target duration.
- [x] Keep task cards focused on timer actions and move makeup/postponement into `更多` and task detail.
- [x] Add optional completion reflection panel with save-and-complete and skip-and-complete actions.
- [x] Implement reflection entry as a bottom sheet that preserves timer/task state when cancelled.
- [x] Add task-detail objective/reflection display and side-effect-free reflection editing.
- [x] Add two-line objective collapse/expand behavior with active-task default expansion.

## 6. Verification and Release

- [ ] Add backend tests for state transitions, idempotent rewards, overtime, schedule resolution, motivation length validation, objective validation, and reflection edits.
- [ ] Add frontend checks for timer control labels and picker payloads.
- [x] Run backend tests, frontend/admin type checks, and mini program build.
- [x] Validate OpenSpec change and update task status before deployment.
