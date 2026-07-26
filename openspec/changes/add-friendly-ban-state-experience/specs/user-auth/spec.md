# user-auth Delta Specification

## MODIFIED Requirements

### Requirement: User can login with WeChat
The system SHALL reject an actively banned linked account consistently and SHALL allow an expired ban to clear before normal WeChat authentication or account linking proceeds.

#### Scenario: Active ban during returning WeChat login
- **WHEN** a linked OpenID belongs to an actively banned user
- **THEN** the system returns the canonical HTTP 403 ban envelope without an application JWT

#### Scenario: Active ban during WeChat account link
- **WHEN** valid linking credentials identify an actively banned H5 account
- **THEN** the system returns the canonical HTTP 403 ban envelope without linking the OpenID

#### Scenario: Expired ban during WeChat login
- **WHEN** the linked user's ban deadline has passed
- **THEN** the system clears the ban fields and continues normal login

### Requirement: API requests require authentication
The system SHALL evaluate the current persisted ban status for every authenticated request and SHALL preserve an otherwise valid token while access is paused.

#### Scenario: Protected request during active ban
- **WHEN** a valid application JWT belongs to an actively banned user
- **THEN** the system returns the canonical HTTP 403 ban envelope and no additional user data

#### Scenario: Protected request after timed ban expires
- **WHEN** a valid application JWT calls `/auth/me` after its timed ban deadline
- **THEN** the system clears the expired ban and returns the current user normally

## ADDED Requirements

### Requirement: Ban responses expose safe recovery metadata
The system SHALL return HTTP 403 and code 403 with a friendly message and a `data` object containing only `account_banned=true`, reason, RFC3339 `banned_until`, explicit `permanent`, and RFC3339 `server_now` ban metadata.

#### Scenario: Timed ban response
- **WHEN** any supported login or authenticated path detects an active timed ban
- **THEN** the response marks `permanent=false` and provides the authoritative deadline and server time

#### Scenario: Permanent ban response
- **WHEN** a ban uses the server's permanent sentinel semantics
- **THEN** the response marks `permanent=true` so no client date guess is required

### Requirement: Clients provide a dedicated paused-access experience
The H5 and mini-program clients SHALL centrally persist safe ban metadata, retain an existing token, and route active bans to one dedicated banned page.

#### Scenario: Ban detected from transport or business envelope
- **WHEN** the request wrapper receives HTTP 403 or code 403 with `account_banned=true`
- **THEN** it stores the validated payload, retains the token, routes to the banned page, and rejects the request

#### Scenario: Timed ban displayed
- **WHEN** the banned page has a valid timed-ban payload
- **THEN** it displays the reason fallback, absolute unlock time, and live server-aligned days/hours/minutes/seconds countdown

#### Scenario: Permanent ban displayed
- **WHEN** the banned payload marks the ban permanent
- **THEN** the page displays permanent wording without a countdown

#### Scenario: Retained token recovers
- **WHEN** `/auth/me` succeeds after a timed ban expires or an administrator unbans the user
- **THEN** the client clears ban state and routes through the normal user route decision

#### Scenario: Login-time ban has no token
- **WHEN** login returns a ban before an application token exists
- **THEN** the page offers return-to-login retry rather than claiming it can refresh authenticated status

#### Scenario: Repeated blocked refresh
- **WHEN** a refresh from the banned page receives another ban response
- **THEN** the wrapper updates state without re-launching the same page or creating a route loop
