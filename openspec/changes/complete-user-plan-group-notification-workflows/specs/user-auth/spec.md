# user-auth Delta Specification

## ADDED Requirements

### Requirement: User controls sessions and password
The system SHALL support current-device logout, authenticated password change, and one-time administrator-assisted password reset.

#### Scenario: Logout current device
- **WHEN** the user chooses logout
- **THEN** the client removes the current access token without deleting account data or other sessions

#### Scenario: Change password
- **WHEN** the authenticated user supplies the correct current password and a valid new password
- **THEN** the system stores a new hash, invalidates prior tokens, and returns a fresh token

#### Scenario: Redeem reset code
- **WHEN** the user submits the correct username, an unused unexpired reset code, and a valid new password
- **THEN** the system resets the password, consumes the code, and invalidates prior tokens

#### Scenario: Administrator creates reset code
- **WHEN** an administrator requests account recovery for an active user
- **THEN** the system returns a random reset code once that expires after 30 minutes and records the action
