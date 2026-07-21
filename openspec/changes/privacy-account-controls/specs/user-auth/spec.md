## ADDED Requirements

### Requirement: User can view masked phone number
The system SHALL show the bound phone number in masked form in account settings.

#### Scenario: View masked phone
- **WHEN** user opens account settings
- **THEN** system shows phone number as masked text such as `138****5678`

### Requirement: User can rebind phone number
The system SHALL allow users to replace their bound phone number through WeChat phone authorization.

#### Scenario: Rebind phone
- **WHEN** user completes phone authorization for a new phone number
- **THEN** system replaces the old bound phone number and records the account event

### Requirement: User can deactivate account with data choice
The system SHALL let users deactivate their mini program account and choose whether to retain or delete historical data.

#### Scenario: Deactivate and retain data
- **WHEN** user chooses to deactivate while retaining data
- **THEN** system preserves user plans, tasks, check-ins, sessions, slack records, and group history for future restoration

#### Scenario: Deactivate and delete data
- **WHEN** user chooses to deactivate and delete data
- **THEN** system deletes or anonymizes personal and learning data according to the documented deletion policy

#### Scenario: Restore retained account
- **WHEN** user logs in again with the same verified identity after retaining data
- **THEN** system restores the user's retained account data

### Requirement: Privacy policy explains data usage
The system SHALL provide privacy policy content covering phone number, avatar storage, learning records, AI usage, group-visible metrics, notifications, admin access, and account deletion choices.

#### Scenario: User opens privacy policy
- **WHEN** user opens the privacy policy entry
- **THEN** system displays what data is collected, why it is used, and how the user can deactivate or delete data
