## MODIFIED Requirements

### Requirement: User can deactivate account with data choice
The system SHALL let users deactivate their mini program account and choose whether to retain or delete historical data, and SHALL allow an authorized administrator to invoke the same delete-and-anonymize policy for a normal user without allowing administrator account deletion.

#### Scenario: Deactivate and retain data
- **WHEN** user chooses to deactivate while retaining data
- **THEN** system preserves user plans, tasks, check-ins, sessions, slack records, and group history for future restoration

#### Scenario: Deactivate and delete data
- **WHEN** user chooses to deactivate and delete data
- **THEN** system deletes or anonymizes personal and learning data according to the documented deletion policy

#### Scenario: Administrator deletes normal account
- **WHEN** an authorized administrator requests deletion of a normal user account
- **THEN** the system applies the same deletion and anonymization policy
- **AND** the deleted account cannot authenticate with its former credentials

#### Scenario: Administrator targets an administrator account
- **WHEN** an authorized administrator requests deletion of themselves or another administrator account
- **THEN** the system rejects the request without deleting or anonymizing the target

#### Scenario: Restore retained account
- **WHEN** user logs in again with the same verified identity after retaining data
- **THEN** system restores the user's retained account data

## ADDED Requirements

### Requirement: User can manage login username with a monthly limit
The system SHALL allow an authenticated active user to change their login username to a unique 4-24 character ASCII letter, digit, or underscore value, including an 11-digit mobile number, at most three times in a Shanghai calendar month.

#### Scenario: User changes login username
- **WHEN** an authenticated user selects a valid unused login username
- **THEN** the system updates the username and normalized username, records the change, invalidates other sessions, and returns a refreshed session token

#### Scenario: User selects a mobile number as login username
- **WHEN** an authenticated user selects a valid 11-digit mobile number as the login username
- **THEN** the system accepts it as a login identifier without claiming that the phone number has been verified

#### Scenario: User exceeds monthly change limit
- **WHEN** an authenticated user has already changed the login username three times in the current Shanghai calendar month
- **THEN** the system rejects another change without changing the username

#### Scenario: User selects an already used login username
- **WHEN** an authenticated user selects a username used by another active account
- **THEN** the system rejects the change with a conflict response
