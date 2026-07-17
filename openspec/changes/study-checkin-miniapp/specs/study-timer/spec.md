## ADDED Requirements

### Requirement: User can start learning on schedule
The system SHALL allow users to begin a study session when a scheduled task starts.

#### Scenario: Start learning
- **WHEN** user taps "开始学习" on a daily task
- **THEN** system records actual_start timestamp and sets task status to in_progress

### Requirement: User can pause and resume learning
The system SHALL allow users to temporarily pause the timer and resume later.

#### Scenario: Pause and resume
- **WHEN** user taps "暂停" during study
- **THEN** system pauses the timer, records the session up to that point
- **WHEN** user taps "恢复学习"
- **THEN** system starts a new StudySession

### Requirement: User can end learning
The system SHALL allow users to stop the timer and record the session.

#### Scenario: End learning
- **WHEN** user taps "结束学习"
- **THEN** system records actual_end, calculates study_minutes, saves the StudySession

### Requirement: User can extend learning beyond schedule
The system SHALL allow users to continue studying after the scheduled end time.

#### Scenario: Extend learning
- **WHEN** user taps "延长学习" after scheduled_end is reached
- **THEN** system continues timing and marks the task as overtime

### Requirement: User can temporarily leave and resume on same day
The system SHALL allow users to pause study for urgent matters and resume the same day.

#### Scenario: Pause for urgent matter
- **WHEN** user taps "暂停学习" with reason (e.g. "接电话")
- **THEN** system pauses timer, records intermediate session
- **WHEN** user taps "恢复学习"
- **THEN** system resumes timing, total still counts toward daily goal

### Requirement: 23:30 boundary reminder
The system SHALL notify users before midnight if learning is still active.

#### Scenario: 23:30 warning
- **WHEN** current time is 23:30 and user has active (in_progress) tasks
- **THEN** system sends a WeChat message asking user to decide: extend to tomorrow or stop now

### Requirement: User can make up end time
The system SHALL allow users to retroactively record the end time if they forgot to stop the timer.

#### Scenario: Makeup recording
- **WHEN** user sends PUT /api/tasks/:id/makeup with end_time and reason
- **THEN** system calculates study_minutes from actual_start to provided end_time

### Requirement: User can postpone today's task to another day
The system SHALL allow users to defer an unfinished task to a specific future date and time.

#### Scenario: Postpone to tomorrow
- **WHEN** user taps "推到明天" and selects time (e.g. 21:00)
- **THEN** system creates PostponeRecord and reschedules the task

#### Scenario: Postpone with date selection
- **WHEN** user taps "延期" and picks target_date and target_time
- **THEN** system reschedules the task to the chosen slot

### Requirement: Overtime study counts toward statistics
The system SHALL include overtime study minutes in all statistical calculations.

#### Scenario: Overtime counted
- **WHEN** user studies 5h on a 4h-scheduled day
- **THEN** stats show 5h studied, with 1h marked as overtime

### Requirement: Early finish allowed
The system SHALL allow users to mark a task complete before the scheduled end time if all content is done.

#### Scenario: Early complete
- **WHEN** user finishes all task content before scheduled_end
- **THEN** user can tap "提前完成" and the task is marked completed
