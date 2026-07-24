## MODIFIED Requirements

### Requirement: User authenticates with WeChat identity and a unique application nickname
The system SHALL authenticate with WeChat account identity, SHALL require a valid unique application nickname before study features are used, and SHALL NOT require a phone number.

#### Scenario: New user authenticates with WeChat
- **WHEN** code2Session succeeds for a previously unknown openid
- **THEN** the system creates an incomplete user profile and returns `nickname_required: true`
- **AND** routes the user to nickname setup before plan, task, group, or check-in features

#### Scenario: WeChat login does not provide nickname
- **WHEN** the login response contains no reliable nickname
- **THEN** the user explicitly chooses an application nickname
- **AND** the system does not fabricate a public nickname from openid or an internal user ID

#### Scenario: User chooses an available nickname
- **WHEN** the user submits a valid nickname whose normalized key is unused
- **THEN** the backend saves it and unlocks normal study features

#### Scenario: User chooses a duplicate nickname
- **WHEN** the normalized nickname is already used by another active account
- **THEN** the backend returns HTTP 409 and asks the user to choose another nickname
- **AND** does not disclose details of the existing account

#### Scenario: User submits an invalid nickname
- **WHEN** the nickname is outside 2-20 display characters or violates content rules
- **THEN** the backend rejects it with a user-correctable validation message

#### Scenario: Existing user has a blank or duplicate nickname
- **WHEN** a legacy user logs in without a valid unique nickname
- **THEN** the system requires nickname setup before study features

#### Scenario: User logs in without phone number
- **WHEN** WeChat authentication and nickname setup are complete
- **THEN** the user can use all normal study features without phone authorization or phone binding

## REMOVED Requirements

### Requirement: Phone binding is configurable after WeChat login
**Reason**: The product no longer uses phone binding as an identity, onboarding, invitation, or study-access requirement.

**Migration**: Existing stored phone data follows retention/deletion policy but is no longer displayed, requested, or used to gate product features. Phone-binding routes and mini-program entry points are retired after compatibility review.

### Requirement: User can view masked phone number
**Reason**: Phone number is removed from the supported mini-program profile experience.

**Migration**: Remove masked-phone UI and update privacy documentation to reflect that new phone collection has stopped.

### Requirement: User can rebind phone number
**Reason**: Phone-based profile management is outside the revised product scope.

**Migration**: Remove rebinding UI and disable the route after confirming no external client depends on it.

## ADDED Requirements

### Requirement: Nicknames are normalized and unique
The system SHALL persist a normalized nickname key with database-enforced uniqueness and SHALL preserve the user's validated display form.

#### Scenario: Nicknames differ only by case or surrounding whitespace
- **WHEN** two users submit nicknames that normalize to the same key
- **THEN** only one nickname is accepted

#### Scenario: Concurrent nickname claims
- **WHEN** two users concurrently claim the same normalized nickname
- **THEN** the database accepts exactly one and the other receives a conflict response

#### Scenario: User changes nickname
- **WHEN** an authenticated user selects a new valid available nickname
- **THEN** the same validation and uniqueness rules apply
- **AND** existing learning records remain associated with the user
