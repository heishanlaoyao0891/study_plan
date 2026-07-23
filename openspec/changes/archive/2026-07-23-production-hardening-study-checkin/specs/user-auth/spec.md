## MODIFIED Requirements

### Requirement: User can login with WeChat
The system SHALL authenticate users via WeChat login (code → openid exchange) in production, and SHALL allow mock login only when explicitly enabled for local development.

#### Scenario: Successful login
- **WHEN** user taps "微信登录" in the mini program
- **THEN** system exchanges code for openid and returns a JWT token

#### Scenario: Returning user
- **WHEN** user logs in again with the same WeChat account
- **THEN** system returns existing user's data with a new JWT token

#### Scenario: Production login uses WeChat code exchange
- **WHEN** `WECHAT_LOGIN_MOCK=false` and user logs in with a WeChat code
- **THEN** system calls WeChat `jscode2session`, stores or finds the user by openid, and returns a JWT token

#### Scenario: Mock login disabled in production
- **WHEN** `WECHAT_LOGIN_MOCK=false` and a mock-only login code is provided
- **THEN** system rejects the login instead of creating a mock user

#### Scenario: Local mock login enabled
- **WHEN** `WECHAT_LOGIN_MOCK=true` and user logs in locally
- **THEN** system may create a deterministic mock openid for local testing

## ADDED Requirements

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
