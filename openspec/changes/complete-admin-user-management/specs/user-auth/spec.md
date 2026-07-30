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
