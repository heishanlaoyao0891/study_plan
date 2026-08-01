## ADDED Requirements

### Requirement: Authenticated clients receive fail-closed channel features
The system SHALL expose authenticated client feature availability without exposing provider credentials or administration-only configuration, and SHALL report mini-program AI planning as disabled when its setting cannot be loaded.

#### Scenario: Mini-program feature is disabled
- **WHEN** an authenticated supported client loads features while `mini_program_ai_enabled` is false
- **THEN** the response reports mini-program AI planning as unavailable

#### Scenario: Feature configuration cannot be loaded
- **WHEN** the client-feature service cannot load its persisted configuration
- **THEN** it reports mini-program AI planning as unavailable and exposes no AI provider details

### Requirement: Supported clients identify their platform
The shared client request layer SHALL identify supported WeChat mini-program requests as `mp-weixin` and H5 requests as `h5` using channel metadata.

#### Scenario: WeChat mini-program sends an API request
- **WHEN** a supported mini-program build calls the backend
- **THEN** the request includes the `mp-weixin` client platform value

#### Scenario: H5 sends an API request
- **WHEN** the H5 build calls the backend
- **THEN** the request includes the `h5` client platform value
