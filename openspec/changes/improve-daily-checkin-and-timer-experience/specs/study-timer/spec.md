## MODIFIED Requirements

### Requirement: User can start learning on schedule
The system SHALL allow users to start a task at any time while using its planned time range to define target duration, schedule display, and reminders.

#### Scenario: Start learning
- **WHEN** user taps `开始` on a daily task
- **THEN** system records an active StudySession and sets task status to in_progress

#### Scenario: Start outside planned clock range
- **WHEN** user starts earlier or later than planned_start
- **THEN** the countdown starts from the task's full target duration instead of the remaining wall-clock schedule

### Requirement: User can pause and resume learning
The system SHALL pause and resume task timing without completing the task and SHALL accumulate all sessions toward the same task.

#### Scenario: Pause and resume
- **WHEN** user taps `暂停` during study
- **THEN** system closes the active session, preserves accumulated study time, and changes the primary action to `开始`
- **WHEN** user taps `开始` again
- **THEN** system creates a new active StudySession and resumes from the remaining target duration

### Requirement: User can pause or complete a task
The system SHALL treat resumable pause and task-completing end actions as separate user intentions.

#### Scenario: Pause remains resumable
- **WHEN** user taps `暂停` during an active study session
- **THEN** system closes and saves the active StudySession, preserves accumulated study time, and leaves the task incomplete

#### Scenario: End learning before target duration
- **WHEN** user confirms `结束本次学习` before the target duration is reached
- **THEN** system closes any active session and marks the task completed

#### Scenario: Complete achieved task
- **WHEN** user taps `完成任务` after the countdown reaches zero
- **THEN** system closes the active session and marks the task completed

### Requirement: User can extend learning beyond schedule
The system SHALL continue recording overtime after target duration until the user explicitly completes the task.

#### Scenario: Extend learning
- **WHEN** countdown reaches zero while the task remains active
- **THEN** system does not automatically pause or complete the task and the client displays increasing overtime duration

#### Scenario: Achieved-state control
- **WHEN** target duration has been reached
- **THEN** the primary `暂停` action changes to a prominent `完成任务` action

### Requirement: Early finish is explicit
The system SHALL allow users to complete a task before target duration while clearly distinguishing early finish from resumable pause.

#### Scenario: Complete early
- **WHEN** user finishes all task content before target duration
- **THEN** user can select a visually secondary `结束本次学习` action and confirm that it completes the task

#### Scenario: Early finish cancelled
- **WHEN** user opens the `结束本次学习` confirmation and cancels
- **THEN** the active or paused timer state remains unchanged

### Requirement: User can postpone today's task to another day
The system SHALL allow users to defer an unfinished task using bounded date and planned-time controls.

#### Scenario: Postpone to tomorrow
- **WHEN** user selects tomorrow and confirms the default planned time range
- **THEN** system creates PostponeRecord and reschedules the task

#### Scenario: Postpone with date selection
- **WHEN** user opens postponement
- **THEN** the client provides native date, planned-start, and planned-end pickers preloaded from the task

#### Scenario: Postpone with date and time
- **WHEN** user confirms a target date and valid planned time range
- **THEN** system creates a PostponeRecord and updates the task's planned schedule

#### Scenario: Postponement conflicts with another task
- **WHEN** the selected target range conflicts with an existing task
- **THEN** the client presents a conflict confirmation before submitting an explicit override

### Requirement: User can edit makeup start and end time
The system SHALL allow users to correct actual study date, start time, end time, and reason through bounded controls.

#### Scenario: User corrects both start and end
- **WHEN** user selects valid actual date/start/end values and confirms the summary
- **THEN** system recalculates study minutes from the corrected times

#### Scenario: Invalid makeup range rejected
- **WHEN** user submits actual_end earlier than actual_start or an end time in the future
- **THEN** system rejects the makeup request

#### Scenario: Excessive makeup duration rejected
- **WHEN** user submits a corrected session longer than 8 hours
- **THEN** the backend rejects the request

#### Scenario: Makeup picker defaults
- **WHEN** user opens makeup
- **THEN** the client preloads the task date and known actual or planned times and displays the calculated duration before submission

## ADDED Requirements

### Requirement: Task card displays live timing
The mini program SHALL display planned time range, remaining target duration, current accumulated study time, and overtime state for each daily task.

#### Scenario: Task is not active
- **WHEN** task is pending or paused
- **THEN** the card shows persisted accumulated study time and remaining target duration

#### Scenario: Task is active
- **WHEN** task has an active StudySession
- **THEN** the card updates elapsed and remaining time live from server session timestamps

#### Scenario: App returns from background
- **WHEN** the mini program returns to foreground
- **THEN** the client refreshes authoritative task/session state and reconciles the displayed timer

### Requirement: Task controls reflect timer state
The mini program SHALL expose controls whose labels and emphasis match task timer state.

#### Scenario: Pending or paused task
- **WHEN** task has no active session and is incomplete
- **THEN** primary action is `开始` and pre-target `结束本次学习` remains visually secondary

#### Scenario: Active task before target
- **WHEN** task is active with remaining target duration
- **THEN** primary action is resumable `暂停` and task-completing `结束本次学习` remains visually secondary

#### Scenario: Active task after target
- **WHEN** task is active and target duration is reached
- **THEN** primary action is prominent `完成任务` and `结束本次学习` is removed

#### Scenario: Completed task
- **WHEN** task is completed
- **THEN** card shows completed state without start, pause, early-finish, or completion actions

### Requirement: Task card prioritizes the active learning flow
The mini program SHALL keep daily task cards focused on objective, timing, and primary timer controls while placing lower-frequency corrections in secondary navigation.

#### Scenario: User views incomplete task card
- **WHEN** a daily task is incomplete
- **THEN** the card shows task title/status, compact objective, planned range, timing state, primary timer action, and low-emphasis `结束本次学习` where applicable

#### Scenario: User needs makeup or postponement
- **WHEN** user opens `更多` or task detail
- **THEN** the client provides makeup and postponement actions outside the primary card action row

#### Scenario: User views completed task card
- **WHEN** a task is completed
- **THEN** the card collapses to title, total study duration, and completed state and opens full objective/reflection detail on tap
