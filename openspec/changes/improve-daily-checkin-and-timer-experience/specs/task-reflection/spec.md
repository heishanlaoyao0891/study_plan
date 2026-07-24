## ADDED Requirements

### Requirement: New tasks have concrete objectives
The system SHALL require a concise, actionable objective for newly created manual and AI-generated daily tasks.

#### Scenario: User creates a manual task
- **WHEN** user creates a new daily task
- **THEN** the request must include an objective describing what the user is expected to complete

#### Scenario: AI creates daily tasks
- **WHEN** AI returns a plan preview
- **THEN** every task includes an objective that is more specific than merely repeating the task or plan title

#### Scenario: Legacy task has no objective
- **WHEN** user views a legacy task without an objective
- **THEN** the client displays `暂未填写任务目标` without fabricating objective content

#### Scenario: User edits legacy task
- **WHEN** user saves edits to a legacy task without an objective
- **THEN** the client and backend require a concrete objective before saving

#### Scenario: Incomplete task objective is long
- **WHEN** an incomplete task objective exceeds two rendered lines
- **THEN** the card displays two lines with an in-place expand/collapse affordance

#### Scenario: Task is actively timed
- **WHEN** a task has an active study session
- **THEN** its objective defaults to expanded for the current page visit

#### Scenario: Completed task objective is long
- **WHEN** a completed task is shown on the daily page
- **THEN** the card stays compact and the full objective is available in task detail

### Requirement: User can record an optional completion reflection
The system SHALL allow the user to save an optional reflection of at most 500 Chinese-display characters when ending learning early or completing an achieved task.

#### Scenario: Complete with reflection
- **WHEN** user enters a valid reflection and selects `保存心得并完成`
- **THEN** system completes the task and stores the reflection

#### Scenario: Completion panel opens
- **WHEN** user selects `结束本次学习` before the target or `完成任务` after reaching the target
- **THEN** the client opens a bottom-sheet completion panel over the daily page with task summary, actual duration, optional reflection input, and save/skip actions

#### Scenario: Skip reflection
- **WHEN** user selects `跳过，直接完成`
- **THEN** system completes the task without requiring reflection text

#### Scenario: Completion panel cancelled
- **WHEN** user closes or cancels the completion panel
- **THEN** the task remains incomplete and its active or paused timer state is preserved

#### Scenario: Reflection exceeds length limit
- **WHEN** user submits reflection content longer than 500 Chinese-display characters
- **THEN** system rejects the reflection and preserves the unsubmitted completion panel state

### Requirement: User can review and edit completion reflection
The system SHALL display saved reflection in task detail and allow the task owner to add or edit it after completion.

#### Scenario: View completed task
- **WHEN** user opens a completed task with a reflection
- **THEN** task detail displays the task objective, actual study duration, and reflection

#### Scenario: Add reflection after skipping
- **WHEN** user previously skipped reflection and later adds one from task detail
- **THEN** system stores the reflection without changing task or check-in state

#### Scenario: Edit saved reflection
- **WHEN** user updates an existing reflection
- **THEN** system saves the new text without changing study time, completion timestamp, check-in, streak, or rewards

### Requirement: Reflection is private by default
The system SHALL treat completion reflections as private task content unless a future explicit sharing capability is introduced.

#### Scenario: Group member views another member
- **WHEN** a group member views another member's progress
- **THEN** the system does not expose that member's completion reflection
