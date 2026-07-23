## ADDED Requirements

### Requirement: Backend APIs have focused regression tests
The system SHALL include focused backend tests covering authentication, ownership, task lifecycle, check-in rewards, stats, admin authorization, and scheduling edge cases.

#### Scenario: Run backend tests
- **WHEN** developer runs the backend test command
- **THEN** core API behavior is verified against an isolated test database

### Requirement: Database migrations are verified
The system SHALL verify automatic migrations and key constraints used by the application.

#### Scenario: Migration verification
- **WHEN** backend starts with an empty database in test mode
- **THEN** required tables, indexes, and constraints are created successfully

### Requirement: Release has a repeatable verification command
The system SHALL provide a documented local verification flow before release.

#### Scenario: Run release verification
- **WHEN** developer runs the verification command or script
- **THEN** backend tests, backend build, frontend type-check/build, admin console type-check/build if present, and OpenSpec validation complete successfully

### Requirement: Frontend critical flows have a manual preview checklist
The system SHALL maintain a lightweight manual checklist for mini program preview testing.

#### Scenario: Run mini program preview checklist
- **WHEN** developer prepares a release build
- **THEN** checklist covers login/phone binding, plan creation, AI preview commit, task timing, auto check-in, postpone/makeup, and group nudge flows

### Requirement: CI is optional in this iteration
The system SHALL treat local verification as the required baseline and CI as optional future work.

#### Scenario: No CI configured
- **WHEN** project has no GitHub Actions workflow
- **THEN** release verification can still be completed using the local PowerShell verification script
