# user-auth Delta Specification

## MODIFIED Requirements

### Requirement: User can login with WeChat
The system SHALL automatically exchange a WeChat code during mini-program startup, SHALL authenticate a linked account without a login landing action, and SHALL require conditional account setup before a new WeChat identity can access study features.

#### Scenario: Linked returning account
- **WHEN** mini-program startup resolves an OpenID linked to a complete account
- **THEN** the system returns an application JWT and enters the application without requesting account fields or displaying a login action

#### Scenario: New WeChat identity
- **WHEN** an OpenID is not linked to an account
- **THEN** the system returns a short-lived registration token and does not create an orphan user or application JWT

#### Scenario: Create an account from WeChat
- **WHEN** the user submits a valid invitation, unique username, nickname, and password with the registration token
- **THEN** the system atomically creates one account linked to the OpenID and consumes the invitation

#### Scenario: Link an existing H5 account
- **WHEN** a new OpenID submits the correct username and password of an unlinked H5 account
- **THEN** the system links that OpenID to the existing user ID without consuming another invitation

### Requirement: API requests require authentication
The system SHALL accept only application JWTs associated with complete persisted accounts for study APIs.

#### Scenario: Registration token requests study API
- **WHEN** a client sends a WeChat registration token to a study API
- **THEN** the system returns 401 Unauthorized

## ADDED Requirements

### Requirement: User can authenticate on H5
The system SHALL support invitation registration with username, nickname, and password, followed by username/password login.

#### Scenario: H5 registration
- **WHEN** an unused valid invitation and valid unique account fields are submitted
- **THEN** the system creates the shared account, consumes the invitation, and returns an application JWT

#### Scenario: H5 password login
- **WHEN** a user submits the correct username and password
- **THEN** the system returns the same account and data used by its linked WeChat identity

#### Scenario: Invalid credentials
- **WHEN** a username is absent or its password is wrong
- **THEN** the system returns the same generic authentication error

### Requirement: Account creation requires an invitation
The system SHALL allow only administrators to generate invitations and SHALL accept each active invitation once before its seven-day expiry.

#### Scenario: Invitation generated
- **WHEN** an administrator requests one or more invitations
- **THEN** the system creates cryptographically random codes and returns each plaintext code once

#### Scenario: Invitation reused, expired, or disabled
- **WHEN** registration submits an invitation that was used, expired, or disabled
- **THEN** the system rejects registration without creating an account
