# admin-config Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
### Requirement: Admin can configure slack exchange rates
The system SHALL allow administrators to configure how study time converts to slack time.

#### Scenario: Configure global rate
- **WHEN** admin updates global SlackConfig with checkin_minutes, streak_bonus, quality_bonus
- **THEN** all users use the new rate by default

#### Scenario: Configure per-user rate
- **WHEN** admin updates SlackConfig for a specific user
- **THEN** that user uses the custom rate instead of global default

### Requirement: Admin can view user list
The system SHALL allow administrators to view registered users.

#### Scenario: View users
- **WHEN** admin requests GET /api/admin/users
- **THEN** system returns user list with basic info

### Requirement: Slack config has default fallback
The system SHALL apply global defaults when no per-user config exists.

#### Scenario: Fallback to default
- **WHEN** user has no custom SlackConfig entry
- **THEN** system applies the global default SlackConfig

### Requirement: There is exactly one admin
The system SHALL have exactly one admin account. Other users are normal users.

#### Scenario: Admin is unique
- **WHEN** system initializes
- **THEN** the first user or a designated user is marked as admin (role=admin)

### Requirement: Admin can ban users permanently
The system SHALL allow the admin to permanently block a user from logging in.

#### Scenario: Permanent ban
- **WHEN** admin sends POST /api/admin/users/:id/ban with duration_hours=0 and reason
- **THEN** system sets banned_until to a far-future sentinel value and records the reason

### Requirement: Admin can ban users for a limited time
The system SHALL allow the admin to ban a user for a specific duration.

#### Scenario: Timed ban
- **WHEN** admin sends POST /api/admin/users/:id/ban with duration_hours=24 and reason
- **THEN** system sets banned_until to now + 24 hours and records the reason

#### Scenario: 2-day ban
- **WHEN** admin sends POST /api/admin/users/:id/ban with duration_hours=48
- **THEN** system sets banned_until to now + 48 hours

### Requirement: Admin can unban users
The system SHALL allow the admin to lift a ban early.

#### Scenario: Unban
- **WHEN** admin sends POST /api/admin/users/:id/unban
- **THEN** system clears banned_until and banned_reason

### Requirement: Banned users cannot log in
The system SHALL reject login attempts from banned users with a clear message.

#### Scenario: Banned user login attempt
- **WHEN** banned user tries to log in
- **THEN** system returns 403 with banned_reason and banned_until

### Requirement: Expired ban auto-clears on login
The system SHALL automatically unban a user if the ban duration has passed.

#### Scenario: Auto-clear on login
- **WHEN** a timed-banned user logs in after banned_until has passed
- **THEN** system clears banned_until and allows normal login

### Requirement: Admin authentication
The system SHALL restrict admin endpoints to authorized users only.

#### Scenario: Unauthorized access
- **WHEN** non-admin user requests admin endpoint
- **THEN** system returns 403 Forbidden

