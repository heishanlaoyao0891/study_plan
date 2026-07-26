## ADDED Requirements

### Requirement: User can authorize slack balance reminders
The system SHALL expose a separately configurable and separately authorized WeChat reminder when slack balance becomes low or negative.

#### Scenario: Balance crosses the low threshold
- **WHEN** session settlement leaves a previously positive balance at 10 minutes or less
- **THEN** the system attempts one idempotent subscription delivery describing the remaining balance or debt

#### Scenario: Reminder is not configured or authorized
- **WHEN** the administrator template is disabled or the user has no matching authorization
- **THEN** the balance settlement still succeeds and delivery is recorded as skipped
