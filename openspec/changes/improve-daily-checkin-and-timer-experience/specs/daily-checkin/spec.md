## MODIFIED Requirements

### Requirement: Checkin requires all tasks done
The system SHALL treat task completion as the source of truth for check-in eligibility and SHALL require explicit user confirmation to finalize an eligible daily check-in.

#### Scenario: Checkin with incomplete tasks
- **WHEN** user tries to checkin but not all daily tasks are done
- **THEN** system returns an error with count of remaining tasks

#### Scenario: Checkin after all tasks complete
- **WHEN** all daily tasks for a plan are completed
- **THEN** system marks the plan/date as eligible for check-in without automatically finalizing it

#### Scenario: Complete task from mini program
- **WHEN** the user completes the final remaining task for a plan on a date
- **THEN** the backend makes the daily check-in action available
- **AND** the client does not perform a duplicate automatic check-in mutation

#### Scenario: User finalizes eligible check-in
- **WHEN** the user taps `完成今日打卡` after all tasks are complete
- **THEN** the backend finalizes the check-in, updates streak state, and applies rewards idempotently

### Requirement: Completed checkin earns slack time
The system SHALL award slack minutes once when the user explicitly finalizes an eligible daily check-in.

#### Scenario: Earn on checkin
- **WHEN** user finalizes an eligible daily checkin
- **THEN** system adds configured slack_minutes to user's slack_balance

#### Scenario: Repeated check-in request
- **WHEN** the finalized check-in request is retried or repeated
- **THEN** the system returns the existing completed state without awarding slack minutes again

#### Scenario: Task completion triggers check-in reward
- **WHEN** task completion makes the plan/date eligible for check-in
- **THEN** the system waits for explicit user finalization before awarding slack minutes

## ADDED Requirements

### Requirement: Check-in action remains visible
The mini program SHALL keep the daily check-in action visible on the check-in page and communicate its current eligibility.

#### Scenario: Tasks remain incomplete
- **WHEN** one or more daily tasks are incomplete
- **THEN** the check-in action is disabled and displays the remaining task count

#### Scenario: All tasks are complete
- **WHEN** no daily tasks remain incomplete
- **THEN** the check-in action becomes enabled and invites the user to complete the daily check-in

#### Scenario: Check-in already finalized
- **WHEN** the daily check-in is already complete
- **THEN** the action shows the completed state and cannot grant rewards again
