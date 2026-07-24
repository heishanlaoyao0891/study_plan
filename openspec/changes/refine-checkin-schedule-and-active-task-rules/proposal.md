## Why

The daily check-in flow still has several sources of ambiguity. Planned task ranges are treated as conflicting whenever they overlap at all, even though a short transition overlap can be intentional. The backend prevents duplicate active sessions only per task, so a user can start two different tasks at the same time. The disabled check-in label `还剩 1 项` does not explain what remains or why the action is unavailable. The `待处理学习` section actually contains interrupted records requiring user confirmation, while the `连胜天数` metric does not update intuitively after check-in. Task detail also falls back to a generic layout and presents group visibility as an unclear button.

## What Changes

- Allow planned task ranges to overlap while requiring each task's distinct covered duration to remain below 60 minutes.
- Reject plan creation, task splitting, schedule updates, task edits, AI plan generation/commit, and postponement when any affected task accumulates 60 or more covered minutes.
- Enforce one active study session per user across all tasks and plans.
- When another task is active, prevent starting or resuming a task and identify the active task in the response.
- Disable other task start actions in the mini program while one task is active and provide a direct explanation.
- Move check-in out of individual task cards into one page-level daily check-in panel.
- Allow the page-level check-in after the user completes at least one task that day, regardless of how many other tasks remain.
- Rename the interrupted-record section to `正在努力中`, with supporting copy that distinguishes records needing attention from the currently timed task.
- Rename `连胜天数` to `连续打卡` and calculate it from scheduled study days so today's final check-in updates it immediately without losing the previous streak before completion.
- Make task group visibility an understandable, editable switch with ownership and privacy enforcement.
- Redesign task detail using the established warm, cute mini-program visual language and a clear primary-action hierarchy.
- Promote AI plan generation to a dedicated primary entry on the plan page instead of placing it among utility shortcuts.
- Simplify plan-page navigation by removing the account shortcut, demoting settings, and renaming ambiguous recovery copy to `重新安排`.
- Simplify plan-card actions: use `延期` with a day picker and calculated date preview, remove duplicate batch shifting, and move destructive deletion into a low-emphasis more menu.
- Show each plan's task completion percentage directly on its card.
- Let users drag plan actions to choose their order and whether they appear directly or inside `更多`.
- Upgrade `重新安排` from a coarse automatic date spread into an editable, schedule-aware preview that validates before applying.
- Require every user to choose a unique nickname before using study features and replace internal-ID plan invitations with privacy-limited nickname search.
- Remove phone binding from the supported onboarding and study-access flow.

## Capabilities

### Modified Capabilities

- `plan-management`: Replace pairwise overlap warnings with a per-task cumulative covered-duration rule.
- `study-timer`: Enforce a single active task per user and expose the blocking active task.
- `daily-checkin`: Replace plan-linked check-in actions with one user/date daily check-in, clarify interrupted-record terminology, and correct consecutive check-in behavior.
- `plan-management`: Clarify task-level group visibility editing and align task detail with the product visual system.
- `plan-management`: Reorganize the plan page and replace internal-ID invitations with nickname discovery.
- `user-auth`: Require a unique application nickname while removing phone binding from product scope.
- `user-onboarding`: Make nickname setup the required post-login step and prioritize AI-assisted first-plan creation.

## Impact

- Backend schedule-conflict calculation and all task/plan schedule mutation paths.
- Study-session indexes, start/resume transactions, and active-session response contracts.
- Mini program check-in, plan, AI preview, schedule, and task-detail controls.
- Login/onboarding, user schema, nickname update/search endpoints, invitation privacy, and rate limiting.
- Backend consecutive check-in calculation and task visibility update validation.
- Migration and concurrency tests for user-level active-session uniqueness.

## Confirmed Decisions

- For each task, merge all portions of its planned interval covered by other same-date tasks and measure the union duration.
- A task with 0 through 59 covered minutes is valid; a task with 60 or more covered minutes is invalid.
- The overlap limit applies to planned ranges on the same calendar date, not to actual study-session timestamps.
- Adjacent ranges have zero overlap and remain valid.
- The limit is authoritative in the backend and cannot be bypassed with an overload or conflict confirmation flag.
- Every schedule mutation validates all affected tasks, not only the newly created or edited task.
- In the A/B/C example where A overlaps B for 30 distinct minutes and C overlaps B for another 30 distinct minutes, B is invalid at 60 minutes while A and C remain valid at 30 minutes each.
- A user may have only one active StudySession across every task and plan.
- Starting or resuming another task does not automatically pause or complete the current task.
- The API returns the blocking task ID and title so the client can explain the conflict and navigate to it.
- Daily check-in is a page-level action and MUST NOT be rendered inside or visually attached to any task card.
- Completing at least one task on the selected date unlocks `完成今日打卡`; before that, the panel says `完成任意 1 个任务后即可打卡`.
- Other unfinished tasks do not block daily check-in and are not described as remaining check-in requirements.
- A user/date can be checked in and rewarded only once, independent of the number of plans or tasks.
- The section title is `正在努力中`; its subtitle explains interrupted or cross-day records that need attention, while the active timer remains labeled `学习中` and is not duplicated there.
- The header metric is named `连续打卡`, measured in eligible scheduled study days rather than competitive "wins" or raw calendar days.
- If today has not been checked in yet, the API returns the consecutive count completed through the latest prior study day; completing one task and finalizing today's page-level check-in increments it immediately.
- Task owners may edit task-level group visibility even for legacy tasks without being forced to edit unrelated objective content.
- Group-visible tasks expose only the fields already allowed by group privacy rules; completion reflections remain private.
- Task detail follows the existing warm gradient, rounded card, soft shadow, pill, and playful accent system instead of a generic utility-page layout.
- `AI 生成计划` is the dominant plan-page call to action; manual creation remains available but visually secondary.
- The primary plan-page tool row contains only high-frequency planning destinations such as `日程`, `小组`, `提醒`, and `重新安排`; settings moves to a low-emphasis overflow or page footer, and account/phone entry is removed.
- Plan-card `平移任务` becomes `延期`; the user chooses a positive number of days and sees the calculated old/new date summary before confirmation. The separate `批量平移` action is removed.
- Delete is available only from a low-emphasis `更多` menu and still requires a destructive confirmation.
- WeChat authorization identifies the account but does not reliably provide a nickname; first use therefore requires the user to choose an application nickname.
- Nicknames are normalized and unique across active accounts. Phone number is neither requested nor required for login, onboarding, invitation, or normal study use.
- Invitation discovery uses rate-limited nickname search with exact and fuzzy matching; it does not expose internal user IDs or a browseable newest-user directory.
- Plan completion is `completed_tasks / total_tasks` for the plan, displayed with counts and a progress bar; a plan with no tasks shows `尚未生成任务` instead of a misleading 0%.
- Plan-card action layout is user-managed: the first two enabled actions are shown directly, remaining actions appear in `更多`, and drag editing controls order and placement.
- `小组内可见` is not a plan-card action. It is an explicit privacy switch in plan/task detail with explanatory copy and side-effect-free saving.
- `重新安排` shows overdue tasks, proposed dates/times, conflicts, and skipped study days before anything changes; the user can edit or deselect rows and apply only a validated preview.
