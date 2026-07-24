# Tasks

## 1. Contract and Migration

- [x] Add shared API error payloads for per-task excessive covered duration and a blocking active task.
- [x] Define per-task intersection-union calculation with an exclusive 60-minute limit.
- [x] Add migration preflight for users with multiple open sessions.
- [x] Replace the task-level active-session unique index with a user-level partial unique index.
- [x] Add a user/date daily check-in model with a unique index and one reward marker.
- [x] Migrate completed legacy plan/date check-ins into daily records without issuing rewards.

## 2. Planned Schedule Validation

- [x] Add a shared same-date validator that merges each task's covered intervals and validates all affected tasks.
- [x] Enforce the per-task 60-minute exclusive limit in manual plan creation and schedule-template updates.
- [x] Enforce the limit in manual task creation, task schedule edits, postponement, and batch shifts.
- [x] Enforce the limit while splitting one plan into multiple generated tasks.
- [x] Add the cumulative covered-duration rule and A/B/C example to AI-agent planning prompts.
- [x] Enforce deterministic validation in AI preview generation, refinement, and commit against preview and persisted tasks.
- [x] Keep weekly-hours and active-plan warnings overridable while making excessive overlap non-overridable.
- [x] Return every invalid task with covered minutes and merged covered intervals to clients.

## 3. Single Active Task Backend

- [x] Change start/resume lookup and insertion to enforce one active session per user transactionally.
- [x] Keep repeated start/resume for the same active task idempotent.
- [x] Return HTTP 409 with active task/session metadata when another task is running.
- [x] Preserve the existing active task and requested task unchanged on conflict.
- [x] Update midnight compensation and recovery paths to remain compatible with user-level uniqueness.

## 4. Mini Program Experience

- [x] Detect the currently active task from authoritative timer responses.
- [x] Disable start/resume controls for all other tasks and explain `另一任务学习中` with the active task title.
- [x] Refresh timer state after a backend active-task conflict and provide navigation to the running task.
- [x] Remove check-in controls and eligibility copy from every task card.
- [x] Add one dedicated page-level daily check-in panel outside the task list.
- [x] Show `完成任意 1 个任务后即可打卡` at zero completed tasks, `完成今日打卡` when eligible, and `今日已打卡` after finalization.
- [x] Show completed-task progress without presenting unfinished tasks as check-in requirements.
- [x] Run the same per-task covered-duration validation in plan, task, AI, and postponement forms before submission.
- [x] Show every invalid task, covered duration, and covered intervals in correction messages.
- [x] Rename `待处理学习` to `正在努力中` and explain which records need time confirmation or schedule adjustment.
- [x] Keep the currently running task in the normal `学习中` card and exclude it from interrupted records.
- [x] Rename the header metric from `连胜天数` to `连续打卡`.

## 5. Consecutive Check-in

- [x] Replace current-active-plan streak calculation with persisted scheduled-study-day calculation in Asia/Shanghai.
- [x] Preserve the completed-through-yesterday count while today's daily check-in remains unfinished.
- [x] Increment immediately after one completed task unlocks and the user finalizes today's daily check-in.
- [x] Skip dates without generated tasks and break on missed past qualifying dates.
- [x] Return explicit consecutive-checkin fields and update group metrics to use the same calculation.

## 6. Task Detail and Privacy

- [x] Add a focused owner-only task visibility endpoint or update mode that changes only `public_to_group`.
- [x] Allow visibility changes for legacy tasks without requiring unrelated objective edits.
- [x] Keep reflection and other private task content out of group-facing responses.
- [x] Replace the ambiguous visibility button with an optimistic switch, explanatory copy, save feedback, and failure rollback.
- [x] Redesign task detail with warm hero, status pill, timer summary, objective card, compact facts, and anchored primary action.
- [x] Move postponement, makeup, visibility, and other secondary actions out of the equal-weight button grid.
- [x] Keep completed-task detail focused on duration, objective, reflection, and history.

## 7. Plan Page Information Architecture

- [x] Add a prominent standalone AI plan-generation hero and a secondary manual-create action.
- [x] Reduce the primary tool row to `日程`, `小组`, `提醒`, and `重新安排`.
- [x] Remove the plan-page account/phone-binding shortcut and demote settings to overflow or footer.
- [x] Rename recovery navigation and page copy to `重新安排` while preserving overdue-task preview behavior.
- [x] Add plan task totals, completed counts, completion percentage, and progress bars; show a taskless state without 0%.
- [x] Add default plan actions with two direct slots and an always-accessible `更多`/`编辑操作` path.
- [x] Add drag editing between `直接显示` and `更多操作`, including order changes and a two-direct-action display cap.
- [x] Persist one global per-user plan-action layout with allowlisted action IDs and forward-compatible defaults.
- [x] Replace `平移任务` with `延期`, a positive day selector, and calculated old/new date summary.
- [x] Remove the separate `批量平移` card action and retain scoped shifting only for internal recovery if needed.
- [x] Keep delete muted in either placement and require a final destructive confirmation.
- [x] Keep `小组内可见` out of card actions and render it as an explanatory detail-page switch.
- [x] Show `重新安排` only with overdue work and include its overdue count.
- [x] Upgrade rearrangement preview with plan study days, resolved times, overlap validation, row editing/deselection, and stale-state protection.
- [x] Apply rearrangement only to validated owned unfinished non-active tasks and return applied/skipped summaries.

## 8. Nickname Identity and Invitation

- [x] Add normalized nickname storage and database uniqueness with a legacy-user migration audit.
- [x] Define nickname length, Unicode normalization, reserved-term, contact-info, and moderation validation.
- [x] Return `nickname_required` after login and gate study routes until profile setup completes.
- [x] Add nickname setup/change API and first-use mini-program screen with conflict feedback.
- [x] Remove phone binding from onboarding, plan navigation, account UI, and study-feature authorization.
- [x] Review phone binding/rebinding compatibility and retire all mini-program entry points while retaining backend compatibility temporarily.
- [x] Add authenticated rate-limited nickname search with minimum query length, ranking, result cap, and minimal fields.
- [x] Replace numeric-ID invite UI with debounced nickname search, candidate confirmation, and friendly empty states.
- [x] Do not expose newest registered users, registration timestamps, openid, phone, role, or learning data.
- [x] Recheck plan ownership, self-invite, membership, account status, and duplicate invitation on submit.

## 9. Verification

- [ ] Add covered-duration boundary tests for 0, 30, 59, 60, and 61 minutes.
- [ ] Add interval-union tests proving duplicate coverage of the same clock minutes is counted once.
- [ ] Add A/B/C tests proving the middle task can fail while the outer tasks remain valid.
- [ ] Add mutation-path tests for plan defaults, task splitting, overrides, task edits, postponement, batch shifts, and AI previews/commits.
- [ ] Add concurrent start tests proving one active session per user and independent sessions for different users.
- [ ] Add frontend checks for disabled task controls and singular/plural check-in copy.
- [ ] Add page-level check-in tests for zero completed tasks, one completed with others pending, all completed, retries, and multiple plans.
- [ ] Add migration and reward-idempotency tests for legacy plan/date check-ins.
- [ ] Add consecutive-checkin tests for incomplete today, completed today, rest days, missed study days, multiple plans, and historical plan changes.
- [ ] Add `正在努力中` tests proving a normal active task is not duplicated in that section.
- [ ] Add visibility authorization and side-effect tests, including legacy tasks and reflection privacy.
- [ ] Add task-detail visual/state checks for pending, running, achieved, and completed tasks.
- [ ] Add plan-page hierarchy checks proving AI is prominent and utility/account/settings buttons are reduced.
- [ ] Add plan completion aggregate tests for partial, complete, and taskless plans.
- [ ] Add action-layout persistence, drag ordering, two-slot cap, unknown-ID, and delete-placement tests.
- [ ] Add delay picker/payload tests and verify the duplicate batch action is absent.
- [ ] Add rearrangement tests for study days, resolved times, AI fallback, overlap conflicts, editable rows, stale previews, and active-task rejection.
- [ ] Add nickname normalization, uniqueness, concurrent claim, legacy profile, and route-gate tests.
- [ ] Add nickname-search ranking, minimum length, rate limit, privacy-field, and invitation authorization tests.
- [ ] Add onboarding checks proving nickname is required and phone binding is absent.
- [x] Run backend tests, frontend/admin type checks, mini program build, and strict OpenSpec validation.
