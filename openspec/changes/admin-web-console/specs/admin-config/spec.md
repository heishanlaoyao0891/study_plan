## MODIFIED Requirements

### Requirement: Admin can configure slack exchange rates
The system SHALL allow administrators to configure how study time converts to slack time through the PC admin console and authorized admin APIs.

#### Scenario: Configure global rate from PC admin console
- **WHEN** admin updates global SlackConfig in the PC admin console
- **THEN** all users use the new rate by default

#### Scenario: Configure per-user rate from PC admin console
- **WHEN** admin updates SlackConfig for a specific user in the PC admin console
- **THEN** that user uses the custom rate instead of global default

### Requirement: Admin can view user list
The system SHALL allow administrators to view registered users in the PC admin console.

#### Scenario: View users in PC admin console
- **WHEN** admin opens the user list in the PC admin console
- **THEN** system returns user list with basic info

#### Scenario: View user role and status
- **WHEN** admin opens a user detail view in the PC admin console
- **THEN** system shows the user's role, ban status, plan count, check-in summary, and slack balance

### Requirement: Admin can ban users permanently
The system SHALL allow the admin to permanently block a user from logging in through the PC admin console and authorized admin APIs.

#### Scenario: Permanent ban from PC admin console
- **WHEN** admin submits a permanent ban with reason in the PC admin console
- **THEN** system sets banned_until to a far-future sentinel value and records the reason

### Requirement: Admin can ban users for a limited time
The system SHALL allow the admin to ban a user for a specific duration through the PC admin console and authorized admin APIs.

#### Scenario: Timed ban from PC admin console
- **WHEN** admin submits a timed ban with duration and reason in the PC admin console
- **THEN** system sets banned_until to now plus the selected duration and records the reason

### Requirement: Admin can unban users
The system SHALL allow the admin to lift a ban early through the PC admin console and authorized admin APIs.

#### Scenario: Unban from PC admin console
- **WHEN** admin clicks unban in the PC admin console
- **THEN** system clears banned_until and banned_reason

## ADDED Requirements

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
