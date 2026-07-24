## ADDED Requirements

### Requirement: Daily check-in is a page-level action
The mini program SHALL present one daily check-in action outside the task cards and SHALL unlock it after the user completes at least one task for the selected date.

#### Scenario: No task is completed
- **WHEN** the user has completed zero tasks for the selected date
- **THEN** the page-level check-in action is disabled and displays `完成任意 1 个任务后即可打卡`

#### Scenario: One task is completed and others remain
- **WHEN** the user has completed at least one task and one or more other tasks remain unfinished
- **THEN** the page-level check-in action is enabled and displays `完成今日打卡`
- **AND** unfinished tasks do not block check-in

#### Scenario: All tasks are complete
- **WHEN** every task is completed but the daily check-in is not finalized
- **THEN** the same page-level action remains enabled and displays `完成今日打卡`

#### Scenario: Check-in is finalized
- **WHEN** the user/date daily check-in is completed
- **THEN** the page-level action displays `今日已打卡` and remains non-repeatable

#### Scenario: Task card is rendered
- **WHEN** any pending, running, paused, achieved, or completed task card is shown
- **THEN** the card contains no daily check-in action or check-in prerequisite copy

#### Scenario: Page shows daily progress context
- **WHEN** the page-level check-in panel summarizes eligibility
- **THEN** it may display `今日已完成 X 个任务`
- **AND** it does not use `还剩 N 项` or imply every task must be completed

### Requirement: Daily check-in is unique per user and date
The system SHALL finalize at most one daily check-in per user/date and SHALL award its reward at most once.

#### Scenario: Eligible user finalizes daily check-in
- **WHEN** the user has at least one completed task on the date and submits the page-level check-in
- **THEN** the backend creates the user/date daily check-in and grants the configured reward once

#### Scenario: Ineligible user attempts check-in
- **WHEN** the user has no completed task on the date
- **THEN** the backend rejects finalization without creating a check-in or reward

#### Scenario: User retries daily check-in
- **WHEN** the same user/date request is repeated or arrives concurrently
- **THEN** all successful responses refer to the same daily check-in and no duplicate reward is granted

#### Scenario: User has tasks from multiple plans
- **WHEN** the user completes one task belonging to any owned plan
- **THEN** the single page-level daily check-in becomes eligible
- **AND** no separate plan-level check-in action is required

#### Scenario: Legacy plan check-ins are migrated
- **WHEN** historical data contains one or more completed plan/date check-ins for a user/date
- **THEN** migration produces at most one completed daily check-in for that user/date
- **AND** migration does not issue a new reward

### Requirement: `正在努力中` uses supportive and accurate states
The mini program SHALL group interrupted tasks requiring correction or rescheduling under `正在努力中` and SHALL distinguish them from the task that is currently timing.

#### Scenario: Midnight-closed task needs confirmation
- **WHEN** a task was auto-closed or marked `needs_decision`
- **THEN** the page lists it under `正在努力中`
- **AND** explains that the previous study record needs a time confirmation or schedule adjustment

#### Scenario: Task is currently running normally
- **WHEN** a task has an open StudySession and does not require a historical decision
- **THEN** it appears as `学习中` in the normal task list
- **AND** it is not duplicated under `正在努力中`

#### Scenario: No records need confirmation
- **WHEN** no interrupted task requires user input
- **THEN** the `正在努力中` section is hidden

## MODIFIED Requirements

### Requirement: Consecutive check-in reflects scheduled study days
The system SHALL report `连续打卡` as consecutive successful scheduled study days and SHALL update the metric immediately after today's unique daily check-in is finalized.

#### Scenario: User finalizes today's daily check-in
- **WHEN** the user has 3 consecutive successful study days through yesterday, completes at least one task today, and finalizes today's daily check-in
- **THEN** the consecutive check-in count becomes 4 in the response shown after the mutation

#### Scenario: Today is not yet fully checked in
- **WHEN** the user has a completed sequence through yesterday but today's daily check-in is not finalized
- **THEN** the header keeps showing the streak completed through yesterday
- **AND** it does not reset to zero merely because today is still in progress

#### Scenario: Other tasks remain after daily check-in
- **WHEN** the user finalizes today's daily check-in after completing one task while other tasks remain unfinished
- **THEN** today's date counts once toward consecutive check-in
- **AND** completing the remaining tasks does not increment it again

#### Scenario: Date has no scheduled tasks
- **WHEN** a calendar date has no generated daily tasks for the user
- **THEN** that date is skipped and does not increase or break the consecutive check-in count

#### Scenario: User missed a prior scheduled study day
- **WHEN** a past date had scheduled tasks but its daily check-in was not finalized
- **THEN** that missed qualifying date breaks the sequence

#### Scenario: Plan configuration changed after historical check-ins
- **WHEN** the user's current active-plan set differs from the plans that had tasks on a historical date
- **THEN** historical consecutive check-in is calculated from persisted tasks and the unique user/date daily check-in

#### Scenario: Header displays the metric
- **WHEN** the check-in page loads the consecutive count
- **THEN** the header label is `连续打卡`
- **AND** it does not use the competitive term `连胜天数`
