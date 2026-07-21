# user-auth Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
### Requirement: User can login with WeChat
The system SHALL authenticate users via WeChat login (code → openid exchange).

#### Scenario: Successful login
- **WHEN** user taps "微信登录" in the mini program
- **THEN** system exchanges code for openid and returns a JWT token

#### Scenario: Returning user
- **WHEN** user logs in again with the same WeChat account
- **THEN** system returns existing user's data with a new JWT token

### Requirement: API requests require authentication
The system SHALL reject unauthenticated API requests.

#### Scenario: Request without token
- **WHEN** client sends a request without Authorization header
- **THEN** system returns 401 Unauthorized

#### Scenario: Request with invalid token
- **WHEN** client sends a request with an expired or invalid JWT
- **THEN** system returns 401 Unauthorized

### Requirement: User has a role
The system SHALL support user roles (normal user, admin). There is exactly one admin.

#### Scenario: Admin role
- **WHEN** admin user logs in
- **THEN** JWT contains role=admin for accessing admin endpoints

### Requirement: Banned users cannot authenticate
The system SHALL check ban status during login.

#### Scenario: Banned user blocked
- **WHEN** a user with banned_until > now attempts to log in
- **THEN** system returns 403 Forbidden with banned_reason and banned_until timestamp

#### Scenario: Ban expired
- **WHEN** a user with banned_until < now attempts to log in
- **THEN** system clears banned_until and allows normal login

