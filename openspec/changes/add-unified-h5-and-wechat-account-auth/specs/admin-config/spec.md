# admin-config Delta Specification

## ADDED Requirements

### Requirement: Administrator manages registration invitations
The system SHALL let an authenticated administrator generate, inspect, and disable early-access invitation codes.

#### Scenario: Bulk generation
- **WHEN** the administrator requests between 1 and 100 invitations
- **THEN** the system returns that many single-use codes expiring seven days later

#### Scenario: Invitation list
- **WHEN** the administrator opens invitation management
- **THEN** the system shows safe code prefixes, creation and expiry times, status, and the account that redeemed each used invitation

#### Scenario: Disable invitation
- **WHEN** the administrator disables an unused active invitation
- **THEN** subsequent registration attempts with that invitation are rejected
