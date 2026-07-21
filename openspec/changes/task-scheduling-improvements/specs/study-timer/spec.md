## MODIFIED Requirements

### Requirement: User can postpone today's task to another day
The system SHALL allow users to defer an unfinished task to a specific future date and planned time range.

#### Scenario: Postpone with date and time
- **WHEN** user selects a target date, planned start time, and planned end time for an unfinished task
- **THEN** system creates a PostponeRecord and updates the task's planned schedule

### Requirement: Overtime study counts toward statistics
The system SHALL include overtime study minutes and manually corrected after-midnight minutes in statistics while attributing after-midnight correction to the previous day's task.

#### Scenario: After-midnight correction counted for previous day
- **WHEN** user corrects a previous day's task end time from 00:00 to 00:30
- **THEN** system attributes the extra 30 minutes to the previous day's task and stats remain consistent

## ADDED Requirements

### Requirement: User can view upcoming task schedule
The system SHALL provide a future 7-day schedule view of planned and completed tasks.

#### Scenario: View future 7-day schedule
- **WHEN** user opens the schedule view
- **THEN** system shows planned tasks, planned time ranges, completion state, and study minutes for today plus the next 6 days

### Requirement: Task has planned time separate from actual time
The system SHALL store planned start/end time separately from actual study session timestamps.

#### Scenario: Planned and actual time differ
- **WHEN** user starts a task later than planned
- **THEN** system preserves both planned time and actual start time

### Requirement: Plan is separate from task
The system SHALL distinguish plans as long-term goal containers and tasks as concrete scheduled execution units.

#### Scenario: User views plan
- **WHEN** user opens a plan
- **THEN** system shows plan-level goal information and associated tasks without treating the plan itself as a timed study session

#### Scenario: User starts learning
- **WHEN** user starts learning
- **THEN** system starts timing a specific task under a plan

### Requirement: Active sessions auto-close at midnight
The system SHALL not automatically continue active study sessions into the next calendar day.

#### Scenario: Session still active at midnight
- **WHEN** current time reaches 00:00 and a task is still in progress
- **THEN** system closes the active session at 00:00, attributes minutes to the previous day's task, and marks the task as needing user decision

#### Scenario: User studied past midnight
- **WHEN** user later edits the task end time to a time after 00:00
- **THEN** system keeps the task associated with the previous day and recalculates study minutes

#### Scenario: User opens app after missed midnight job
- **WHEN** user opens the mini program and a previous active task should have been closed at midnight
- **THEN** frontend-triggered compensation causes system to close or surface the task as needing decision

### Requirement: User can edit makeup start and end time
The system SHALL allow users to correct actual start and actual end time for a task.

#### Scenario: User corrects both start and end
- **WHEN** user edits actual_start and actual_end for a task
- **THEN** system recalculates study minutes from the corrected times

#### Scenario: Invalid makeup range rejected
- **WHEN** user submits actual_end earlier than actual_start or an end time in the future
- **THEN** system rejects the makeup request

#### Scenario: Excessive makeup duration rejected or warned
- **WHEN** user submits a corrected session longer than 8 hours
- **THEN** system rejects it or requires explicit user correction according to implementation policy

### Requirement: User can batch shift future tasks
The system SHALL allow users to shift future tasks in a plan by a number of days.

#### Scenario: Shift future tasks
- **WHEN** user shifts future tasks by 3 days
- **THEN** system moves future task dates by 3 days while preserving planned time ranges where possible

#### Scenario: Shift starts from selected date
- **WHEN** user selects a batch shift start date
- **THEN** system shifts unfinished tasks on or after that date and does not move completed tasks

### Requirement: Scheduling uses product timezone
The system SHALL use `Asia/Shanghai` for date-bound scheduling behavior.

#### Scenario: Midnight boundary evaluated
- **WHEN** backend evaluates 23:30 reminder or 00:00 auto-close
- **THEN** system uses `Asia/Shanghai` date and time

### Requirement: Task completion can auto-complete check-in
The system SHALL automatically complete a plan/date check-in when all tasks for that plan/date are completed.

#### Scenario: Single task completed
- **WHEN** user completes the only task for a plan on a date
- **THEN** system completes that plan's check-in for the date and applies rewards idempotently

#### Scenario: Multiple tasks completed
- **WHEN** user completes the final unfinished task for a plan on a date
- **THEN** system completes that plan's check-in for the date
