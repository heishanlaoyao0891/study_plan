## MODIFIED Requirements

### Requirement: Admin can view user list
The system SHALL allow administrators to view registered users in a Chinese, operator-friendly admin console, including distinct login username, nickname, and OpenID identity fields and searches across those fields.

#### Scenario: View normal users by default
- **WHEN** admin requests GET /api/admin/users without a status or with a blank status
- **THEN** system returns only active accounts that are not currently banned, including login username, nickname, and OpenID

#### Scenario: Filter users by account state
- **WHEN** admin selects all, normal, banned, or deleted state in the PC admin console
- **THEN** the system returns all accounts, active unbanned accounts, active banned accounts, or deleted accounts respectively

#### Scenario: Search users by account identity
- **WHEN** admin requests GET /api/admin/users with a search value matching username, nickname, or OpenID
- **THEN** system returns matching user accounts

#### Scenario: View users in PC admin console
- **WHEN** admin opens the user list in the PC admin console
- **THEN** the console shows distinct login username, nickname, and OpenID columns, defaults to normal accounts, offers all/normal/banned/deleted filters, and provides a clear add-user entry to the invitation workflow

#### Scenario: View user role and status
- **WHEN** admin opens a user detail view in the PC admin console
- **THEN** system shows the user's role, ban status, plan count, check-in summary, and slack balance

#### Scenario: View users with Chinese admin UI
- **WHEN** an administrator opens the user-management page
- **THEN** the page shows Chinese navigation, filters, table headers, status labels, and action labels
- **AND** the visual layout supports quick scanning on PC screens

### Requirement: Admin actions are audited
The system SHALL record sensitive admin actions for later review, including user deletion.

#### Scenario: Admin bans a user
- **WHEN** admin bans a user
- **THEN** system records an audit log with admin id, target user id, action type, reason, and timestamp

#### Scenario: Admin deletes a user
- **WHEN** admin deletes a normal user through the authorized admin API
- **THEN** system records an audit log with admin id, target user id, action type, reason, and timestamp
