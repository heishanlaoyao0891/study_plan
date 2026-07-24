## MODIFIED Requirements

### Requirement: System controls planned schedule overlap and overload
The system SHALL require each task's distinct planned interval covered by other same-date tasks to total less than 60 minutes, SHALL reject a resulting schedule when any affected task reaches 60 covered minutes, and SHALL continue warning about non-time-based workload risks.

#### Scenario: Planned ranges do not overlap
- **WHEN** one task is planned for 19:00-20:00 and another for 20:00-21:00 on the same date
- **THEN** the system accepts the schedules with zero overlap

#### Scenario: One task has less than one hour covered
- **WHEN** task A covers 30 minutes of task B and no other task covers another part of B
- **THEN** the system accepts both tasks because each has fewer than 60 covered minutes

#### Scenario: Middle task reaches one hour cumulatively
- **WHEN** task A covers one 30-minute portion of task B and task C covers a different 30-minute portion of task B
- **THEN** the system rejects the resulting schedule because B has 60 covered minutes
- **AND** reports B as invalid while A and C each remain within their 30-minute budgets

#### Scenario: Multiple tasks cover the same portion
- **WHEN** tasks A and C both cover the same 30-minute portion of task B
- **THEN** the system merges the duplicate covered interval and counts 30 covered minutes for B
- **AND** does not count the same clock minutes twice

#### Scenario: Task remains just below the threshold
- **WHEN** the merged covered intervals of a task total 59 minutes
- **THEN** the system accepts the resulting schedule

#### Scenario: Task exceeds the threshold
- **WHEN** the merged covered intervals of a task total more than 60 minutes
- **THEN** the system rejects the mutation and returns the invalid task's covered intervals and total minutes

#### Scenario: User confirms an excessive overlap
- **WHEN** any task reaches 60 covered minutes and the request includes an overload or conflict confirmation
- **THEN** the system still rejects the schedule because the overlap threshold is not overridable

#### Scenario: Plan template makes an existing task invalid
- **WHEN** a default range, weekday override, or date override causes either a new or existing task to reach 60 covered minutes
- **THEN** the backend rejects plan creation or schedule update before generating invalid tasks

#### Scenario: AI plan contains excessive cumulative coverage
- **WHEN** an AI plan preview or commit causes any generated or existing same-date task to reach 60 covered minutes
- **THEN** validation rejects the preview or commit and does not persist the plan

#### Scenario: AI agent prepares a schedule
- **WHEN** the AI agent generates or refines task times
- **THEN** its prompt states the per-task distinct covered-duration rule and the A/B/C example
- **AND** deterministic validation checks the generated schedule before it is shown or committed

#### Scenario: Weekly workload is excessive without forbidden overlap
- **WHEN** every task has fewer than 60 covered minutes but total weekly hours or active plan count exceeds configured guidance
- **THEN** the system may return the existing overridable workload warning

### Requirement: User can override overload warnings
The system SHALL allow users to override soft workload warnings but SHALL NOT allow users to override the maximum planned-overlap rule.

#### Scenario: Confirm soft workload override
- **WHEN** the system warns only about weekly hours or active plan count and the user confirms
- **THEN** the system creates or updates the plan

#### Scenario: Attempt to override overlap limit
- **WHEN** any task has 60 or more covered minutes
- **THEN** confirmation flags do not bypass the validation error

## ADDED Requirements

### Requirement: Task owner can control group visibility independently
The system SHALL allow a task owner to edit task-level group visibility without changing unrelated task content or learning state.

#### Scenario: Owner makes a task visible
- **WHEN** the task owner enables `小组内可见`
- **THEN** the backend updates only `public_to_group`
- **AND** the detail page immediately reflects the saved state

#### Scenario: Owner makes a task private
- **WHEN** the task owner disables `小组内可见`
- **THEN** the task is removed from task-level group visibility without deleting personal history

#### Scenario: Legacy task lacks an objective
- **WHEN** the owner changes only group visibility on a legacy task with an empty objective
- **THEN** the focused visibility mutation succeeds without requiring an unrelated objective edit

#### Scenario: Non-owner changes visibility
- **WHEN** another user attempts to modify task visibility
- **THEN** the backend rejects the request and preserves the existing value

#### Scenario: Visibility mutation is side-effect-free
- **WHEN** group visibility changes
- **THEN** timing, sessions, completion, reflection, check-in, consecutive check-in, slack balance, and rewards remain unchanged

#### Scenario: Group member views a visible task
- **WHEN** a group member views information permitted by a task's group-visible state
- **THEN** the response excludes the owner's private completion reflection

### Requirement: Task detail uses the shared warm learning design
The mini program SHALL present task detail with the established warm, cute visual language and SHALL prioritize the current learning action over secondary maintenance actions.

#### Scenario: User opens an active task
- **WHEN** the task timer is running or achieved
- **THEN** the page shows a warm hero summary, prominent live timing, objective, and one visually dominant pause or complete action
- **AND** correction and privacy actions do not compete with the primary timer control

#### Scenario: User opens a pending or paused task
- **WHEN** the task is incomplete without an active session
- **THEN** the page emphasizes start or resume and presents schedule, objective, and accumulated time in rounded content cards

#### Scenario: User opens a completed task
- **WHEN** the task is completed
- **THEN** the page emphasizes completion summary, actual duration, objective, reflection, and history without showing timer-start controls

#### Scenario: User views secondary actions
- **WHEN** the user needs postponement, makeup, visibility, or destructive actions
- **THEN** those controls appear in lower-emphasis sections, a `更多` action, or focused bottom sheets
- **AND** the page does not show an equal-weight dense three-column action grid

#### Scenario: Detail page matches core tabs
- **WHEN** the detail page is compared with check-in and plan pages
- **THEN** it uses compatible warm gradients, large rounded cards, soft shadows, coral/pink accents, status pills, spacing, and friendly copy

### Requirement: Plan page prioritizes AI-assisted creation
The mini program SHALL present AI plan generation as the prominent creation path and SHALL keep utilities visually subordinate to core planning work.

#### Scenario: User opens the plan page
- **WHEN** the plan page loads
- **THEN** a prominent `AI 生成计划` action appears separately from utility shortcuts
- **AND** it explains that AI can split a learning goal into scheduled tasks

#### Scenario: User prefers manual creation
- **WHEN** the user does not want AI generation
- **THEN** a visible but secondary `手动创建` action opens the manual plan form

#### Scenario: User views planning tools
- **WHEN** the plan tools are shown
- **THEN** the primary compact tools are `日程`, `小组`, `提醒`, and `重新安排`
- **AND** `恢复` is not used for overdue-task rescheduling

#### Scenario: User looks for account or settings
- **WHEN** the plan page renders primary tools
- **THEN** it does not show an account/phone-binding shortcut
- **AND** settings appears only as a subdued overflow or footer entry

### Requirement: Plan card actions use clear hierarchy
The mini program SHALL show plan completion progress and SHALL let the user manage the order and placement of plan-card actions.

#### Scenario: Plan has generated tasks
- **WHEN** a plan has 8 generated tasks and 3 are completed
- **THEN** the card displays `完成 3/8`, `38%`, and a matching progress bar

#### Scenario: Plan has no generated tasks
- **WHEN** a plan has zero tasks
- **THEN** the card displays `尚未生成任务`
- **AND** does not display a misleading `0%` completion rate

#### Scenario: User views an active plan card
- **WHEN** the plan card is rendered
- **THEN** the first two actions in the user's direct-action layout are shown
- **AND** other enabled actions are available through `更多`

#### Scenario: User edits plan-card actions
- **WHEN** the user opens `编辑操作`
- **THEN** the sheet shows `直接显示` and `更多操作` sections
- **AND** supports dragging actions between sections and reordering within each section

#### Scenario: User saves action layout
- **WHEN** the user confirms a valid arrangement
- **THEN** the same global layout applies to every plan card and persists across devices

#### Scenario: Direct section contains more than two actions
- **WHEN** the saved direct order includes more than two enabled actions
- **THEN** only the first two render directly and the remainder are accessible in `更多`

#### Scenario: Default action layout
- **WHEN** the user has not customized actions
- **THEN** `暂停/恢复` and `编辑` display directly
- **AND** `延期`, `邀请学习伙伴`, and `删除` appear in `更多`

#### Scenario: User delays a plan
- **WHEN** the user selects `延期`
- **THEN** the client provides a bounded positive day selector
- **AND** shows the calculated current and resulting date range before confirmation

#### Scenario: Delay is confirmed
- **WHEN** the user confirms a valid N-day delay
- **THEN** eligible future unfinished tasks move by N days after overlap validation
- **AND** the response summarizes the moved task count and resulting date range

#### Scenario: User looks for batch shifting
- **WHEN** plan actions are shown
- **THEN** no separate `批量平移` button is displayed

#### Scenario: User deletes a plan
- **WHEN** the user opens `更多`
- **THEN** delete appears as a low-emphasis destructive item
- **AND** requires a final confirmation that explains affected tasks and check-in history

#### Scenario: User places delete directly
- **WHEN** the user drags `删除` into direct actions
- **THEN** it remains visually muted and still requires destructive confirmation

#### Scenario: User accesses privacy settings
- **WHEN** the user changes `小组内可见`
- **THEN** the setting appears as an explanatory switch in plan or task detail
- **AND** it is not treated as a customizable plan-card action

### Requirement: User can safely rearrange overdue tasks
The system SHALL provide an editable, validated preview before moving overdue unfinished tasks.

#### Scenario: User has overdue tasks
- **WHEN** one or more owned unfinished tasks are dated before today
- **THEN** the plan page shows `重新安排` with the overdue task count

#### Scenario: User has no overdue tasks
- **WHEN** no task is eligible for rearrangement
- **THEN** `重新安排` is omitted from the core tool row

#### Scenario: User opens rearrangement preview
- **WHEN** the user selects `重新安排`
- **THEN** the system proposes dates and planned times using each plan's study days, overrides, defaults, and existing future tasks
- **AND** writes no changes before confirmation

#### Scenario: User edits the preview
- **WHEN** the user changes a proposed date/time or deselects a task
- **THEN** the client recalculates and displays covered-duration conflicts before apply

#### Scenario: AI is available
- **WHEN** the configured AI provider successfully proposes a rearrangement
- **THEN** deterministic ownership, state, date, and overlap validation runs before the preview is accepted

#### Scenario: AI is unavailable
- **WHEN** AI generation fails or is disabled
- **THEN** a rule-based preview uses selected study days and resolved time ranges and passes the same validation

#### Scenario: Preview becomes stale
- **WHEN** a task is completed, started, deleted, or rescheduled after preview creation
- **THEN** apply rejects or skips the stale row with an explanation and does not overwrite newer state

#### Scenario: User applies valid rows
- **WHEN** the user confirms selected valid proposals
- **THEN** the backend moves only owned unfinished non-active tasks transactionally
- **AND** returns applied and skipped counts

### Requirement: User can find invitation targets by nickname
The system SHALL allow a plan owner to find an eligible user by unique nickname without requiring knowledge of an internal user ID.

#### Scenario: User opens plan invitation
- **WHEN** the owner selects `邀请学习伙伴`
- **THEN** a search sheet prompts `输入昵称查找学习伙伴`
- **AND** does not request a numeric user ID

#### Scenario: User searches by partial nickname
- **WHEN** the owner enters at least 2 nickname characters
- **THEN** the backend returns a limited list ranked by exact, prefix, then substring match
- **AND** each result contains only nickname, optional avatar, and an opaque invitation target identifier

#### Scenario: Search query is too short
- **WHEN** the query has fewer than 2 normalized characters
- **THEN** the client does not search and asks for more characters

#### Scenario: No nickname matches
- **WHEN** no eligible user matches the query
- **THEN** the sheet shows a private empty state without listing latest registered users

#### Scenario: Owner confirms invitation
- **WHEN** the owner selects and confirms a candidate
- **THEN** the backend invites that user after rechecking ownership and eligibility

#### Scenario: Search is abused
- **WHEN** a user sends excessive nickname searches
- **THEN** the backend rate limits requests without exposing additional account information
