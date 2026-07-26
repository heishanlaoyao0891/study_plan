## ADDED Requirements

### Requirement: Administrator can view an operational dashboard
The administrator console SHALL present top-level operational metrics and four visual analyses using server-owned definitions.

#### Scenario: Administrator opens operations overview
- **WHEN** an authenticated administrator opens the overview
- **THEN** the page shows normal-user accounts, 7-day active users, active plans, today's study minutes, and today's check-in users
- **AND** renders user health, 30-day registrations, 30-day learning activity, and plan status charts

#### Scenario: Users are segmented
- **WHEN** user health is calculated
- **THEN** every normal user belongs to exactly one of deleted, inactive, banned, 7-day active, 8-30-day general, or over-30-day zombie segments
- **AND** status and active-ban precedence apply before login recency

### Requirement: Administrator can inspect user login identity
The system SHALL record the latest successful normal-user login and SHALL expose operational identity fields only through administrator user management.

#### Scenario: User receives a session token
- **WHEN** H5 or WeChat authentication successfully issues a user token
- **THEN** the system records the login time and login method

#### Scenario: Administrator views users
- **WHEN** the administrator opens user management
- **THEN** each row displays OpenID, latest login time and method, account status, and slack balance
