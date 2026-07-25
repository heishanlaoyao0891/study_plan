# user-onboarding Delta Specification

## MODIFIED Requirements

### Requirement: New user receives guided first-run onboarding
The system SHALL persist onboarding state per account and onboarding version.

#### Scenario: Complete onboarding
- **WHEN** a user completes onboarding on either client
- **THEN** later logins on H5, mini program, or another device skip onboarding

#### Scenario: Skip onboarding
- **WHEN** a user skips onboarding
- **THEN** the system persists `skipped` and does not show that onboarding version on later logins
