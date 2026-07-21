# daily-checkin Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
### Requirement: User can view today's checkin status
The system SHALL return the completion status of all plans for a given date.

#### Scenario: View today's checkins
- **WHEN** user requests GET /api/checkins?date=2026-07-17
- **THEN** system returns each plan with its completed status for that date

### Requirement: User can toggle checkin for a plan
The system SHALL allow users to mark a plan as completed for a given date.

#### Scenario: Complete a plan
- **WHEN** user sends POST /api/checkins with plan_id and date
- **THEN** system marks the plan as completed for that date

#### Scenario: Uncheck a plan
- **WHEN** user sends POST /api/checkins with plan_id, date, and completed=false
- **THEN** system removes or sets the checkin as not completed

### Requirement: Checkin requires all tasks done
The system SHALL only allow marking a plan as checked in when all daily tasks for that plan are completed.

#### Scenario: Checkin with incomplete tasks
- **WHEN** user tries to checkin but not all daily tasks are done
- **THEN** system returns an error with count of remaining tasks

#### Scenario: Checkin after all tasks complete
- **WHEN** all daily tasks for a plan are completed
- **THEN** system automatically or manually allows the daily checkin

### Requirement: Completed checkin earns slack time
The system SHALL award slack minutes to the user upon successful daily checkin.

#### Scenario: Earn on checkin
- **WHEN** user completes a daily checkin
- **THEN** system adds configured slack_minutes to user's slack_balance

