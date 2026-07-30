## ADDED Requirements

### Requirement: Authentication page exposes invitation request contact
The system SHALL show users without an invitation a low-emphasis contact path that opens the administrator's QR code from a stable static asset.

#### Scenario: Visitor needs an invitation
- **WHEN** an unauthenticated visitor opens the authentication page without an invitation
- **THEN** the page shows an administrator contact entry below the main authentication card
- **AND** activating the entry opens the administrator QR code for invitation requests

#### Scenario: QR code can be replaced without code changes
- **WHEN** the release operator provides the administrator's personal QR-code image
- **THEN** placing it at `/static/invite-qrcode.png` updates the QR code shown by the authentication page

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

#### Scenario: Link despite legacy incomplete OpenID holder
- **WHEN** an unresolved identity submits the correct username and password of an H5 account and the current OpenID is held only by a legacy incomplete account without username, nickname, or password credentials
- **THEN** the system clears the OpenID from the incomplete holder, marks that holder inactive, links the current OpenID to the H5 account, and returns an application JWT without consuming another invitation

#### Scenario: Linked identity conflict is clear
- **WHEN** an unresolved identity tries to link an H5 account but the current OpenID is already attached to another complete account
- **THEN** the system rejects the request with a Chinese conflict message explaining that the WeChat identity is already bound

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
