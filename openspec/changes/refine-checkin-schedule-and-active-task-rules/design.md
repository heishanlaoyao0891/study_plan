## Context

Planned schedules and actual study sessions represent different constraints. Planned ranges help users organize their day and can tolerate a small transition overlap. Actual sessions represent time being actively recorded and must be mutually exclusive for trustworthy statistics. The current implementation checks schedule conflicts as warnings and uses a partial unique index scoped to `task_id`, which does not prevent two different tasks from running for one user.

## Goals / Non-Goals

**Goals:**

- Permit small planned overlaps while rejecting schedules that are not realistically executable.
- Make user-level active study state authoritative and race-safe.
- Make daily check-in a page-level milestone that unlocks after one completed task.
- Apply the same rules consistently across manual, AI, edit, and postponement flows.
- Make interrupted-record and consecutive-check-in concepts match their actual backend semantics.
- Bring task detail and privacy editing into the established mini-program visual and interaction system.
- Make the plan page read as a core planning workspace instead of a grid of unrelated shortcuts.
- Provide a usable invitation identity model without depending on phone numbers or silent WeChat nickname access.

**Non-Goals:**

- No automatic pause, stop, or task switching when another task is started.
- No configurable per-user overlap threshold in this change.
- No restriction requiring actual study to occur inside the planned range.
- No changes to the explicit daily check-in reward model.
- No phone-number binding, rebinding, or phone-based user discovery.
- No public directory or newest-user feed.

## Decisions

### Planned overlap is budgeted per task

For a task interval `T = [start, end)`, the validator intersects `T` with every other task interval on the same date, merges those intersection intervals, and measures the duration of their union. This is the task's `covered_minutes`. Merging prevents one clock minute from being counted twice when multiple tasks overlap the same portion of `T`.

`covered_minutes` must be strictly less than 60. Values from 0 through 59 are valid; 60 or more is a hard validation error. For example, if A covers the first 30 minutes of B and C covers a different 30 minutes of B, B has 60 covered minutes and is rejected. A and C each have only 30 covered minutes and remain valid. If A and C cover the same 30-minute portion of B, B has 30 covered minutes rather than 60.

The backend evaluates resolved task ranges by concrete date. Weekday templates and date overrides are resolved before comparison. A task is not compared with itself during edits. Every proposed mutation validates the complete resulting schedule for each affected date because adding or changing one task can make an existing task exceed its overlap budget. Completed historical tasks are preserved, but mutations that introduce or increase a future conflict must satisfy the new rule.

### The threshold is a hard rule, not an overload warning

`confirm_overload` and `confirm_conflict` continue to apply to soft workload warnings where relevant, but they cannot override `covered_minutes >= 60`. Validation responses identify every invalid task and include its merged covered intervals and total covered minutes so the client can present a useful correction message.

### Validation runs in frontend, backend, and AI planning

The frontend uses the same interval-union rule to give immediate feedback while users create plans, split tasks, edit schedules, or postpone work. Frontend validation improves usability but never replaces backend enforcement.

The backend applies the authoritative validator after resolving the proposed schedule and before any write. Manual plan creation, plan template updates, manual task creation, task splitting, task schedule edits, postponement, batch shifts, AI preview acceptance, and AI commit all use the same rule.

The AI planning prompt explicitly states that every task must have fewer than 60 distinct covered minutes and includes the A/B/C counterexample. Provider output is parsed and validated deterministically; invalid output is rejected or regenerated before preview and cannot be committed by client-side edits that violate the rule.

### One active session is enforced per user

The database partial unique index changes from active session uniqueness by task to active session uniqueness by user for rows where `end_time IS NULL`. Start and resume execute the active-session lookup and insert in one transaction. A uniqueness conflict is resolved by loading the user's active session and returning a domain conflict rather than a generic database error.

If the active session belongs to the requested task, start/resume remains idempotent and returns that session. If it belongs to another task, the API returns HTTP 409 with `active_task_id`, `active_task_title`, and `active_session_id`. The existing task is not mutated.

### The client mirrors but does not replace backend enforcement

The daily page derives the active task from returned timer views. Start/resume controls for other tasks are disabled and labeled `另一任务学习中`. Tapping the disabled context or conflict prompt identifies the active task and can navigate to its detail. Lifecycle refresh remains necessary because another device may change active state.

### Daily check-in is a page-level milestone

The check-in page contains one dedicated daily check-in panel outside the task list. Task cards contain task state and timer actions only; they never contain a check-in button or check-in eligibility copy.

Eligibility is intentionally a minimum daily-start standard: the authenticated user must complete at least one task associated with the selected Asia/Shanghai date. The count or state of other tasks does not block check-in. The panel states are:

- No task completed: disabled action `完成任意 1 个任务后即可打卡`
- At least one task completed: enabled action `完成今日打卡`
- Daily check-in finalized: completed state `今日已打卡`

The panel may show context such as `今日已完成 X 个任务`, but MUST NOT describe unfinished tasks as check-in requirements. `remaining_tasks` may remain on task/plan summaries for progress display, but it is not an eligibility field and is not rendered as `还剩 N 项` beside a task.

### Daily check-in is unique by user and date

The new source of truth is one daily check-in record per `(user_id, date)`, independent of plan ID. Explicit finalization verifies `completed_task_count >= 1` in the same transaction, creates or returns the unique record, and awards slack exactly once. Retries are idempotent.

Existing plan/date check-in rows are migration inputs only. A historical user/date is considered checked in if at least one legacy completed check-in exists for that date. Migration creates at most one daily record and preserves a single reward marker without issuing new rewards. New clients and consecutive-checkin calculation do not require every plan to have a check-in row.

### `正在努力中` is supportive but state-accurate

`pending-decision` contains tasks whose previous timing record was interrupted, auto-closed, or otherwise requires correction. The section is named `正在努力中`, with explanatory copy such as `这些学习还需要你确认时间或调整安排`. The friendly title does not replace explicit per-record state labels. A task with a real open session is represented by its `学习中` timer card and the single-active-task state.

The query contract must avoid mixing a current open session into the interrupted-record section merely because its task status is `in_progress`. The backend returns only records that require a user decision, or adds an explicit decision reason/state that lets the client separate them safely.

### Consecutive check-in follows scheduled study days

The metric is renamed from `连胜天数` to `连续打卡`. A qualifying day is a date on which the user had one or more generated daily tasks. The day is successful when the user completed at least one task and explicitly finalized the unique page-level daily check-in. Dates with no scheduled tasks are skipped rather than breaking the sequence.

The calculation starts from today when today's daily check-in is finalized. If today has scheduled work but is not yet checked in, it starts from the latest prior qualifying day, preserving the completed sequence until the day ends. Once the user completes one task and finalizes today's check-in, the response increments immediately. A missed past qualifying day breaks the sequence. Historical evaluation uses persisted daily tasks and user/date check-ins, not the user's current active-plan set.

The API returns `consecutive_checkin_days`, `today_qualified`, and a display string. A temporary `streak` alias may be retained only if an existing shipped client requires it; new UI and tests use the explicit field name.

### Group visibility is a focused privacy mutation

Task detail shows a switch labeled `小组内可见` with supporting text describing what group members can see. Only the task owner can change it. The mutation updates only `public_to_group`; it does not require or modify title, objective, timing, completion, reflection, check-in, streak, or reward state. On failure, the switch returns to its previous value and explains the error.

Completion reflections remain private regardless of the visibility switch. Group APIs continue exposing only explicitly permitted task/progress fields.

### Task detail inherits the product visual language

Task detail is a focused learning surface rather than a generic form. It uses the same warm gradient background, large rounded cards, soft shadows, coral/pink primary accents, status pills, and friendly microcopy established on check-in and plan pages. The information hierarchy is:

1. Hero summary with task title, parent plan, status, and live timer.
2. Objective as the primary content card.
3. Compact schedule and accumulated-time facts.
4. Optional description, completion reflection, privacy, and history sections.
5. A sticky or visually anchored primary timer action with secondary actions moved into `更多` or dedicated sheets.

The page must not render all actions as an equal-weight three-column button grid. Completed tasks prioritize reflection and history; active tasks prioritize timer state; correction actions remain secondary.

### Plan page has one primary creation path

The plan page is the second core product module and uses a deliberate hierarchy:

1. A prominent `AI 生成计划` hero/action below the page summary, with supporting copy describing goal-to-task decomposition.
2. A secondary `手动创建` action for users who already know the exact schedule.
3. A compact tool row for `日程`, `小组`, `提醒`, and `重新安排` when overdue work exists.
4. The plan list as the main page body.
5. Low-frequency settings in a subdued overflow or footer entry, not in the primary tool grid.

The existing `账户` entry is removed from the plan page. The current `恢复` entry is renamed `重新安排` because it previews and moves overdue tasks rather than restoring an account or timer. It appears with an overdue count only when work can be rearranged; otherwise it is omitted from the core tool row.

### `重新安排` is preview-first and schedule-aware

The current implementation is only a basic fallback: it finds unfinished tasks dated before today and assigns them sequentially to tomorrow, the next day, and so on. It is callable, but it does not yet respect study weekdays, resolved plan time ranges, cumulative covered-duration limits, or real AI output. This behavior is insufficient for the promoted product entry.

The revised preview groups overdue tasks by plan and proposes a concrete new date, planned start/end, and reason for each task. It uses the plan's selected weekdays, date overrides, default time range, existing future load, and per-task covered-duration rule. AI may propose the arrangement when configured, but deterministic validation remains authoritative and the rule-based fallback must satisfy the same constraints.

Nothing is written on preview. Users can deselect tasks, edit proposed dates/times, and review conflict messages. Apply accepts a preview version/token plus selected edited actions; the backend reloads owned unfinished tasks, rejects stale or invalid proposals transactionally, and returns moved/skipped counts. Completed tasks and active sessions are never moved.

### Plan cards separate common, secondary, and destructive actions

Plan cards show completion progress before actions. The backend returns `total_tasks`, `completed_tasks`, and integer `completion_rate` for each plan. The card displays `完成 X/Y` and a progress bar. A plan with zero tasks displays `尚未生成任务` and no percentage. Completion uses persisted task status across the plan's generated task set; date filters may be added later but are not silently applied in this change.

`延期` replaces `平移任务`: the user selects a positive integer day count through a bounded picker or stepper, and the client previews the resulting plan/task date range before submission. The backend continues shifting eligible future unfinished tasks by that day count after overlap validation.

The duplicate `批量平移` action is removed. Existing start-date-scoped shift behavior may remain as an internal API for recovery workflows but is not exposed as a second card button. Deletion uses muted styling until the final confirmation sheet, where the destructive consequence is explicit.

### Users customize plan-card actions

`更多` remains as an overflow container, but its contents are not hard-coded. `编辑操作` opens a dedicated reorder sheet. Users drag actions between `直接显示` and `更多操作`, and drag within each section to set order. The first two enabled direct actions render on the card; all remaining enabled actions render in `更多`. This global per-user layout applies consistently to all plan cards so every card remains predictable.

Available customizable actions are `暂停/恢复`, `编辑`, `延期`, `邀请学习伙伴`, and `删除`. Delete may be moved into direct display only with muted destructive styling and still requires confirmation. The system prevents unsupported duplicates and always preserves an accessible `更多`/`编辑操作` path. Defaults are direct `暂停/恢复`, `编辑`; overflow `延期`, `邀请学习伙伴`, `删除`.

Preferences persist server-side as ordered action identifiers and placement, so they survive device changes. Unknown action IDs from a newer client are ignored safely; newly introduced actions default to overflow until the user changes them.

`小组内可见` is deliberately excluded from customizable card actions. Visibility is a privacy setting, represented as a labeled switch in plan detail and task detail, with text explaining which title/objective/progress fields group members can see. Changing it saves immediately, rolls back on failure, and never opens an action workflow.

### Nickname is the required application identity

WeChat `code2Session` provides stable account identity but does not provide a reliable nickname in the current login flow. After successful authentication, a user without a valid nickname is routed to nickname setup before entering study features. The product does not request or gate on phone numbers.

Nickname normalization trims surrounding whitespace, applies Unicode normalization, and compares case-insensitively for uniqueness while preserving the chosen display form. Nicknames are 2-20 display characters, reject control characters, emoji-only values, reserved/admin-like terms, contact information, and abusive content. A database-backed normalized nickname key is unique across active accounts. Nickname changes use the same validation and release policy; deleted-account names require an explicit retention policy before reuse.

Existing users with blank, duplicate, or invalid nicknames are prompted to choose a valid nickname at next login. Migration does not silently append internal IDs to public display names.

### Invitation search is deliberate and privacy-limited

Plan invitation opens a search sheet rather than requesting a numeric user ID. Queries require at least 2 normalized characters and debounce before calling a rate-limited authenticated endpoint. Results rank exact match first, prefix match second, and substring match last. The response contains only opaque invitation target ID, nickname, and optional avatar; it excludes openid, phone, registration time, role, and learning data.

The product does not expose a newest-registration list because it would create a browseable user directory and reveal account activity. Empty search instead explains `输入昵称查找学习伙伴`. The user selects one result, reviews the nickname/avatar, and confirms the invitation. The backend rechecks target eligibility and prevents self-invites, duplicate membership, blocked/deactivated targets, and unauthorized plan access.

## Risks / Trade-offs

- [Existing future schedules contain tasks with 60 or more covered minutes] -> Do not rewrite existing tasks during migration; enforce the rule when affected schedules are created or edited and surface invalid tasks for correction.
- [Pairwise sums double-count the same clock minutes] -> Merge each task's intersection intervals before measuring covered duration.
- [Concurrent starts race] -> Back application checks with the partial unique user-level database index and map constraint conflicts to HTTP 409.
- [SQLite migration cannot replace an index atomically on all deployments] -> Drop the old named index and create the new index during controlled migration after checking for duplicate active sessions.
- [Multiple active sessions already exist] -> Migration preflight identifies affected users; close or flag stale sessions using the existing midnight/decision policy before creating the unique index.
- [Disabled controls feel unresponsive] -> Show the active task title and provide navigation instead of silently ignoring taps.
- [Today's unfinalized check-in resets the visible count to zero] -> Preserve the count through the previous qualifying day until today is finalized or becomes a missed historical day.
- [Current plan edits distort historical streak] -> Derive qualifying dates from persisted daily tasks/check-ins, not the current active-plan list.
- [Visibility toggle leaks private content] -> Keep reflection private and test group response schemas independently from the task owner detail response.
- [Nickname search enables user enumeration] -> Require minimum query length, rate limit, cap results, return minimal fields, and never expose a newest-user directory.
- [Legacy users have duplicate nicknames] -> Gate affected accounts through explicit nickname setup instead of assigning misleading public suffixes.
- [Plan page becomes crowded again] -> Keep one primary AI action, one secondary manual action, four compact tools, and move low-frequency actions to overflow.
- [Custom actions make every card inconsistent] -> Store one global per-user plan-card layout rather than a different layout per plan.
- [Drag customization hides essential access] -> Limit direct actions to two, keep an immutable overflow/edit entry, and validate preference IDs server-side.
- [Recovery preview becomes stale] -> Revalidate task ownership, state, version, dates, and overlap constraints atomically on apply.

## Migration Plan

1. Detect users with more than one open StudySession and resolve stale rows before index creation.
2. Replace the task-scoped active-session partial unique index with a user-scoped index.
3. Add shared per-task interval-union validation and apply it to every schedule mutation path.
4. Deploy backend conflict responses before enabling client-side disabled controls and copy.
5. Verify existing schedules remain readable and unchanged until explicitly edited.
6. Migrate legacy plan/date check-ins into unique user/date daily check-ins without duplicating rewards.
7. Deploy the page-level eligibility and consecutive-checkin response before removing task-card check-in actions.
8. Ship the focused visibility mutation and task-detail redesign together so the switch is both functional and understandable.
9. Add nickname normalization and migration before enforcing the unique index and onboarding gate.
10. Deploy nickname search and invitation confirmation before removing internal-ID invitation UI.
11. Switch plan-page hierarchy and remove phone/account entry after nickname onboarding is available.
12. Add plan progress aggregates and action-layout preferences before replacing the fixed card action row.
13. Upgrade recovery preview/apply validation before promoting `重新安排` conditionally for overdue work.

## API Contract

- Schedule mutations return HTTP 409 when any affected task has `covered_minutes >= 60`. The error data includes `invalid_tasks`, each with `task_id`, `title`, `covered_minutes`, and merged `covered_intervals`, plus `max_covered_minutes_exclusive: 60`.
- `PUT /api/tasks/:id/start` and `/resume` return HTTP 409 when another task is active. The error data includes `active_task_id`, `active_task_title`, and `active_session_id`.
- Repeating start/resume for the already-active task remains successful and returns the existing session.
- `GET /api/checkins/daily?date=YYYY-MM-DD` returns `date`, `completed_task_count`, `eligible`, `completed`, and reward status for the authenticated user/date.
- `POST /api/checkins/daily` accepts `date` and `completed: true`; it succeeds only when at least one owned task on that date is completed and is idempotent per user/date.
- Plan/task list responses may continue returning unfinished counts for progress, but those counts do not control page-level check-in eligibility.
- `GET /api/checkins/consecutive` or the existing streak endpoint returns `consecutive_checkin_days`, `today_qualified`, and `display_text` using Asia/Shanghai scheduled-study dates.
- The interrupted-record endpoint returns only tasks requiring confirmation and includes a machine-readable decision reason when multiple causes are possible.
- The task visibility mutation accepts only `public_to_group` for an owned task and is side-effect-free for all learning, check-in, reflection, and reward state.
- Login response adds `nickname_required`; authenticated study routes reject incomplete profiles with a dedicated profile-setup response until a unique nickname is saved.
- `PUT /api/auth/nickname` accepts a nickname and returns the normalized, saved user profile; uniqueness conflicts return HTTP 409 without revealing the owning account.
- `GET /api/users/search?q=` requires at least 2 characters, returns at most 10 minimal invitation candidates, and is authenticated, rate-limited, and non-browseable without a query.
- Plan invite accepts an opaque target ID selected from search results; raw database IDs are not displayed in the mini program.
- Plan delay accepts a positive `days` value and returns the calculated old/new plan range and moved-task count after validation.
- Plan list responses add `total_tasks`, `completed_tasks`, and `completion_rate`, with `completion_rate` omitted or nullable when no tasks exist.
- `GET/PUT /api/users/me/plan-action-layout` reads and saves ordered direct/overflow action identifiers after allowlist validation.
- Recovery preview returns a version/token and proposed `task_id`, `old_date`, `new_date`, `planned_start`, `planned_end`, `reason`, and validation state; apply requires the preview token and selected actions.
