# user-auth Delta Specification

## MODIFIED Requirements

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

#### Scenario: Local mock login enabled
- **WHEN** `WECHAT_LOGIN_MOCK=true` in local development
- **THEN** the system MAY use a deterministic mock identity while preserving the same automatic-authentication and conditional-setup states

## ADDED Requirements

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
