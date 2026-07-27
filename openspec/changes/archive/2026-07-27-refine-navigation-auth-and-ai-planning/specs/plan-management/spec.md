# plan-management Delta Specification

## ADDED Requirements

### Requirement: Personal utilities are separated from plan management
The client SHALL keep account/data and settings/help entries out of the plan page and SHALL place them in a utility section at the bottom of the rightmost statistics tab.

#### Scenario: User opens plan tab
- **WHEN** a user views the plan list
- **THEN** the page contains plan creation, scheduling, group, reminder, recovery, and plan content without `账号与数据` or `设置与说明` entries

#### Scenario: User reaches the bottom of statistics
- **WHEN** a user scrolls past all statistics content in the rightmost tab
- **THEN** a separated utility section shows `账号与数据` followed by `设置与说明`

#### Scenario: User opens relocated utility
- **WHEN** a user selects either relocated entry
- **THEN** the client opens the existing account/data or settings/help destination without changing its underlying behavior
