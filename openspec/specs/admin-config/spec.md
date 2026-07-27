# admin-config Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
### Requirement: Admin can configure slack exchange rates
The system SHALL allow administrators to configure how study time converts to slack time through the PC admin console and authorized admin APIs.

#### Scenario: Configure global rate
- **WHEN** admin updates global SlackConfig with checkin_minutes, streak_bonus, quality_bonus
- **THEN** all users use the new rate by default

#### Scenario: Configure per-user rate
- **WHEN** admin updates SlackConfig for a specific user
- **THEN** that user uses the custom rate instead of global default

#### Scenario: Configure global rate from PC admin console
- **WHEN** admin updates global SlackConfig in the PC admin console
- **THEN** all users use the new rate by default

#### Scenario: Configure per-user rate from PC admin console
- **WHEN** admin updates SlackConfig for a specific user in the PC admin console
- **THEN** that user uses the custom rate instead of global default

### Requirement: Admin can view user list
The system SHALL allow administrators to view registered users in a Chinese, operator-friendly admin console.

#### Scenario: View users
- **WHEN** admin requests GET /api/admin/users
- **THEN** system returns user list with basic info

#### Scenario: View users in PC admin console
- **WHEN** admin opens the user list in the PC admin console
- **THEN** system returns user list with basic info

#### Scenario: View user role and status
- **WHEN** admin opens a user detail view in the PC admin console
- **THEN** system shows the user's role, ban status, plan count, check-in summary, and slack balance

#### Scenario: View users with Chinese admin UI
- **WHEN** an administrator opens the user-management page
- **THEN** the page shows Chinese navigation, filters, table headers, status labels, and action labels
- **AND** the visual layout supports quick scanning on PC screens

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
The system SHALL allow the admin to permanently block a user from logging in through the PC admin console and authorized admin APIs.

#### Scenario: Permanent ban
- **WHEN** admin sends POST /api/admin/users/:id/ban with duration_hours=0 and reason
- **THEN** system sets banned_until to a far-future sentinel value and records the reason

#### Scenario: Permanent ban from PC admin console
- **WHEN** admin submits a permanent ban with reason in the PC admin console
- **THEN** system sets banned_until to a far-future sentinel value and records the reason

### Requirement: Admin can ban users for a limited time
The system SHALL allow the admin to ban a user for a specific duration through the PC admin console and authorized admin APIs.

#### Scenario: Timed ban
- **WHEN** admin sends POST /api/admin/users/:id/ban with duration_hours=24 and reason
- **THEN** system sets banned_until to now + 24 hours and records the reason

#### Scenario: 2-day ban
- **WHEN** admin sends POST /api/admin/users/:id/ban with duration_hours=48
- **THEN** system sets banned_until to now + 48 hours

#### Scenario: Timed ban from PC admin console
- **WHEN** admin submits a timed ban with duration and reason in the PC admin console
- **THEN** system sets banned_until to now plus the selected duration and records the reason

### Requirement: Admin can unban users
The system SHALL allow the admin to lift a ban early through the PC admin console and authorized admin APIs.

#### Scenario: Unban
- **WHEN** admin sends POST /api/admin/users/:id/unban
- **THEN** system clears banned_until and banned_reason

#### Scenario: Unban from PC admin console
- **WHEN** admin clicks unban in the PC admin console
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
The system SHALL restrict admin endpoints to authorized users only and provide a clear recovery path when an admin session expires.

#### Scenario: Unauthorized access
- **WHEN** non-admin user requests admin endpoint
- **THEN** system returns 403 Forbidden

#### Scenario: Admin session expired
- **WHEN** an admin API request returns unauthorized or forbidden because the session is invalid
- **THEN** the admin console clears the stale session and guides the admin back to the login page

### Requirement: Mini program does not expose admin management
The mini program SHALL NOT expose administrator pages, administrator navigation entries, role management controls, user management controls, or system configuration controls.

#### Scenario: Normal user opens mini program
- **WHEN** any user opens the mini program
- **THEN** no admin management page, role management page, user management entry, or configuration management entry is available in mini program navigation

### Requirement: Admin console is PC-oriented
The system SHALL provide a separate Vue-based PC web console for administrator workflows.

#### Scenario: Admin opens PC console
- **WHEN** an administrator opens the PC admin console and authenticates successfully
- **THEN** the console shows admin workflows such as overview, user management, role/status visibility, ban controls, slack configuration, AI model configuration, subscription message configuration, and audit logs

### Requirement: Admin console uses username and password login
The system SHALL authenticate PC admin console users with username and password for the MVP.

#### Scenario: Admin logs in with valid credentials
- **WHEN** admin submits a valid username and password
- **THEN** system returns an authenticated admin session or JWT with `role=admin`

#### Scenario: Admin login rejects invalid credentials
- **WHEN** admin submits an invalid username or password
- **THEN** system rejects the login without revealing which field was incorrect

#### Scenario: Admin password is stored securely
- **WHEN** admin credentials are persisted
- **THEN** system stores a secure password hash instead of plaintext

### Requirement: Admin actions are audited
The system SHALL record sensitive admin actions for later review.

#### Scenario: Admin bans a user
- **WHEN** admin bans a user
- **THEN** system records an audit log with admin id, target user id, action type, reason, and timestamp

### Requirement: Admin can configure AI model access
The system SHALL allow administrators to configure AI provider settings from the PC admin console.

#### Scenario: Configure AI provider
- **WHEN** admin sets provider, model name, base URL, timeout, daily limit, and enabled state
- **THEN** system saves the AI configuration and uses it for future AI generation requests

#### Scenario: Configure asynchronous Agent budget
- **WHEN** admin selects the background planning Agent budget
- **THEN** the system accepts a 5-minute or 10-minute tier and applies it independently of the short connection-test timeout

#### Scenario: Update AI API key
- **WHEN** admin updates the AI API key
- **THEN** system stores the secret securely and does not return the full key in later API responses

#### Scenario: Disable AI generation
- **WHEN** admin disables AI generation in the PC admin console
- **THEN** user-facing AI generation requests are rejected with a clear disabled message

### Requirement: Admin can configure subscription message templates
The system SHALL allow administrators to configure WeChat subscription message template settings from the PC admin console.

#### Scenario: Configure reminder template IDs
- **WHEN** admin sets template IDs for study start, completion, 23:30 decision, and missed check-in reminders
- **THEN** system saves the template configuration for future notification delivery

#### Scenario: Enable or disable reminder type
- **WHEN** admin toggles a reminder type in the PC admin console
- **THEN** system uses the enabled state when deciding whether to send that reminder

#### Scenario: View notification delivery status
- **WHEN** admin opens subscription message configuration or status view
- **THEN** system shows recent delivery successes and failures without exposing user secrets

### Requirement: Admin console has polished Chinese information architecture
The system SHALL present the PC admin console with Chinese-first navigation, page titles, primary actions, table labels, and form labels.

#### Scenario: Operator scans admin dashboard
- **WHEN** an operator opens the PC admin console
- **THEN** the sidebar, topbar, overview metrics, and page headings use Chinese product terminology
- **AND** the console uses a consistent dashboard style across pages

### Requirement: Admin console preserves operational density without looking unfinished
The system SHALL keep admin pages dense enough for operations while using consistent spacing, cards, buttons, forms, and tables.

#### Scenario: Operator reviews tabular data
- **WHEN** an operator views users, audit logs, suspicious records, feedback, or configuration pages
- **THEN** tables and panels use consistent typography, status treatments, and action affordances

### Requirement: Admin navigation keeps stable dimensions
The PC admin console SHALL keep its sidebar at a stable compact width and SHALL constrain route content to the workspace without allowing overview metrics, charts, tables, loading states, or errors to resize the navigation column.

#### Scenario: Operator opens overview
- **WHEN** an administrator selects `运营总览`
- **THEN** the sidebar width and navigation label layout remain unchanged from other admin routes

#### Scenario: Workspace contains wide content
- **WHEN** an admin route renders content wider than the available workspace
- **THEN** the content wraps, shrinks, or scrolls inside the workspace without widening the sidebar

#### Scenario: Admin viewport narrows
- **WHEN** the PC console is viewed at its supported minimum desktop width
- **THEN** navigation remains usable and workspace content does not overlap or displace it
