## ADDED Requirements

### Requirement: Admin user directory supports complete account operations
The system SHALL allow an authenticated administrator to identify normal user accounts by login username, nickname, and OpenID; search by any of those identifiers; directly create a normal password account; and delete a normal user account from the PC admin console.

#### Scenario: Operator reviews identity columns
- **WHEN** an administrator opens the user directory
- **THEN** each user row shows distinct login username, nickname, and OpenID values

#### Scenario: Operator searches by login username
- **WHEN** an administrator searches the directory with a login username
- **THEN** the matching account is returned

#### Scenario: Operator adds a user
- **WHEN** an administrator selects the user-directory add action
- **THEN** the console accepts a valid login username and nickname, creates a normal account without requiring an invitation, and displays a unique generated initial password only for that successful creation

#### Scenario: Operator deletes a normal user
- **WHEN** an administrator confirms deletion of a normal user in the directory
- **THEN** the system performs the documented account-deletion cleanup, anonymizes the account identity, and records the administrator action

#### Scenario: Operator attempts to delete a protected account
- **WHEN** an administrator attempts to delete themselves or any administrator account
- **THEN** the system rejects the operation without changing the target account
