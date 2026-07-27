# user-auth Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
### Requirement: User can login with WeChat
The system SHALL automatically exchange a WeChat login code during mini-program startup, SHALL authenticate a linked eligible account without presenting a login landing page, and SHALL require conditional account setup before an unresolved OpenID can access study features.

#### Scenario: Linked returning account launches mini program
- **WHEN** the mini program starts and the exchanged OpenID is linked to an active complete account
- **THEN** the system returns an application JWT and routes the user into the application without displaying login, registration, or password-reset choices

#### Scenario: Unresolved WeChat identity launches mini program
- **WHEN** the mini program starts and the exchanged OpenID is not linked to an eligible account
- **THEN** the system returns a short-lived account-setup token without creating an orphan account or authorizing study APIs

#### Scenario: Create an account from WeChat
- **WHEN** an unresolved identity submits a valid launch-carried or manually entered invitation, unique username, nickname, password, and account-setup token
- **THEN** the system atomically creates one account linked to the OpenID, consumes the invitation, and returns an application JWT

#### Scenario: Link an existing H5 account
- **WHEN** an unresolved identity submits the correct username and password of an account that has no other OpenID binding
- **THEN** the system links the current OpenID to that existing user ID and returns an application JWT without consuming another invitation

#### Scenario: Mini-program exchange fails
- **WHEN** WeChat code acquisition or server exchange fails during startup
- **THEN** the client shows a recoverable retry state and does not expose authenticated pages or the H5 authentication selector

#### Scenario: Production login uses WeChat code exchange
- **WHEN** `WECHAT_LOGIN_MOCK=false` and the mini program starts
- **THEN** the system calls WeChat `jscode2session` and never accepts a client-supplied OpenID as proof of identity

#### Scenario: Mock login disabled in production
- **WHEN** `WECHAT_LOGIN_MOCK=false` and a mock-only login code is provided
- **THEN** system rejects the login instead of creating a mock user

#### Scenario: Local mock login enabled
- **WHEN** `WECHAT_LOGIN_MOCK=true` in local development
- **THEN** the system MAY use a deterministic mock identity while preserving the same automatic-authentication and conditional-setup states

### Requirement: H5 authentication actions have conventional hierarchy
The H5 client SHALL present login as the default primary action, registration as a visible secondary action, and password reset as a lower-emphasis link instead of rendering all three as equal tabs.

#### Scenario: H5 visitor opens authentication
- **WHEN** an unauthenticated H5 visitor opens the authentication route
- **THEN** the username/password form and one full-width primary login button receive the strongest visual emphasis
- **AND** `忘记密码？` appears as a small link aligned with the password label
- **AND** the invitation-registration entry appears centered below the primary button

#### Scenario: H5 visitor chooses registration
- **WHEN** the visitor selects the registration action
- **THEN** the same stable-width panel shows invitation, username, nickname, and password fields with one full-width registration button
- **AND** a low-emphasis `已有账号？返回登录` link appears below the primary action

#### Scenario: H5 visitor chooses password reset
- **WHEN** the visitor selects the password-reset link
- **THEN** the same stable-width panel shows username, administrator reset code, and new password fields with one full-width reset button
- **AND** a low-emphasis back-to-login link appears below the primary action
- **AND** reset does not receive equal persistent emphasis to login and registration

#### Scenario: H5 authentication adapts to viewport
- **WHEN** an authentication form is rendered on a narrow mobile or desktop browser viewport
- **THEN** inputs and the primary button remain aligned without overlap
- **AND** auxiliary registration, reset, and return links wrap naturally without becoming peer primary buttons

### Requirement: Mini-program authentication UI is conditional
The mini-program build SHALL exclude the H5 login/register/reset selector and SHALL display account credentials only when automatic OpenID authentication returns an account-setup-required result.

#### Scenario: Returning mini-program user
- **WHEN** automatic startup authentication resolves an existing account
- **THEN** no username, password, invitation, registration, or reset form is shown

#### Scenario: First-use mini-program user
- **WHEN** automatic startup authentication requires account setup
- **THEN** the client shows existing-account linking and invited account creation choices scoped to the short-lived setup token

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

### Requirement: Phone binding is configurable after WeChat login
The system SHALL allow phone number binding after WeChat login, and SHALL only require it before accessing study features when `PHONE_BINDING_REQUIRED=true`.

#### Scenario: User authorizes phone number binding
- **WHEN** user grants phone number authorization in the mini program
- **THEN** system securely stores the verified phone number

#### Scenario: User binds avatar
- **WHEN** user chooses or confirms an avatar in the mini program
- **THEN** system stores the avatar URL or avatar file reference on the user's profile

#### Scenario: User skips phone number binding
- **WHEN** user does not authorize phone number access
- **THEN** system keeps the user logged in and allows study features when `PHONE_BINDING_REQUIRED=false`

#### Scenario: Certified deployment requires phone binding
- **WHEN** `PHONE_BINDING_REQUIRED=true` and user does not authorize phone number access
- **THEN** system blocks study features until phone number binding is completed

#### Scenario: User skips avatar binding
- **WHEN** user does not authorize avatar access
- **THEN** system still allows normal study, plan, timing, and check-in usage when phone binding is not required or phone number is already bound

#### Scenario: User completes phone binding
- **WHEN** user has bound a verified phone number
- **THEN** system allows normal study, plan, timing, and check-in usage

### Requirement: Avatar data is stored outside the database
The system SHALL store avatar binary data outside the relational database and persist only a URL or object-storage key in user profile records. Supported production storage MAY include Tencent Cloud COS or self-hosted MinIO.

#### Scenario: Avatar uploaded
- **WHEN** user uploads or chooses an avatar
- **THEN** system stores the image in object storage or an equivalent file storage layer and saves only its URL or storage key in the database

#### Scenario: Avatar stored in MinIO
- **WHEN** deployment is configured to use self-hosted MinIO for avatar storage
- **THEN** backend uploads avatar files to MinIO and returns only an HTTPS URL or object key to the mini program

### Requirement: User can view masked phone number
The system SHALL show the bound phone number in masked form in account settings.

#### Scenario: View masked phone
- **WHEN** user opens account settings
- **THEN** system shows phone number as masked text such as `138****5678`

### Requirement: User can rebind phone number
The system SHALL allow users to replace their bound phone number through WeChat phone authorization.

#### Scenario: Rebind phone
- **WHEN** user completes phone authorization for a new phone number
- **THEN** system replaces the old bound phone number and records the account event

### Requirement: User can deactivate account with data choice
The system SHALL let users deactivate their mini program account and choose whether to retain or delete historical data.

#### Scenario: Deactivate and retain data
- **WHEN** user chooses to deactivate while retaining data
- **THEN** system preserves user plans, tasks, check-ins, sessions, slack records, and group history for future restoration

#### Scenario: Deactivate and delete data
- **WHEN** user chooses to deactivate and delete data
- **THEN** system deletes or anonymizes personal and learning data according to the documented deletion policy

#### Scenario: Restore retained account
- **WHEN** user logs in again with the same verified identity after retaining data
- **THEN** system restores the user's retained account data

### Requirement: Privacy policy explains data usage
The system SHALL provide privacy policy content covering phone number, avatar storage, learning records, AI usage, group-visible metrics, notifications, admin access, and account deletion choices.

#### Scenario: User opens privacy policy
- **WHEN** user opens the privacy policy entry
- **THEN** system displays what data is collected, why it is used, and how the user can deactivate or delete data
