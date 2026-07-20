## MODIFIED Requirements

### Requirement: User receives reminder when study time arrives
The system SHALL send a WeChat subscription message at the scheduled start time of each task when the user has an active subscription and the template is configured.

#### Scenario: Scheduled start reminder delivered
- **WHEN** it is the task's scheduled_start time and the user has subscribed
- **THEN** system sends a WeChat subscription message and records the delivery result

#### Scenario: User has not subscribed
- **WHEN** a reminder is due but the user has not authorized the corresponding subscription template
- **THEN** system does not send the message and records or reports that subscription permission is missing

### Requirement: User receives completion notification
The system SHALL notify when the daily scheduled study time is completed and record notification delivery status.

#### Scenario: Daily plan completed message delivered
- **WHEN** user's study duration reaches the scheduled_end and the user has subscribed
- **THEN** system sends the completion message and records the delivery result

### Requirement: 23:30 boundary decision reminder
The system SHALL remind users nearing midnight to decide whether to continue, stop, or postpone active study.

#### Scenario: Midnight boundary message delivered
- **WHEN** time is 23:30 and user has in_progress tasks and has subscribed
- **THEN** system sends a decision reminder and records the delivery result

### Requirement: Missed checkin reminder
The system SHALL remind users who have not started their scheduled study after the configured grace period.

#### Scenario: No checkin message delivered
- **WHEN** it is 30 minutes past scheduled_start and user has not started and has subscribed
- **THEN** system sends a missed check-in reminder and records the delivery result
