## ADDED Requirements

### Requirement: Admin independently controls mini-program AI planning
The system SHALL allow an administrator to enable or disable mini-program AI plan availability without changing provider-wide AI enablement or H5 AI availability.

#### Scenario: Admin disables mini-program AI
- **WHEN** an administrator saves AI configuration with mini-program AI disabled
- **THEN** supported mini-program clients receive the disabled feature state while H5 provider behavior is unchanged

#### Scenario: Admin enables mini-program AI
- **WHEN** an administrator explicitly enables mini-program AI after acknowledging the displayed platform-policy warning
- **THEN** supported mini-program clients receive the enabled feature state and the change is recorded in the admin audit log

#### Scenario: New installation has no explicit channel setting
- **WHEN** AI configuration is initialized or migrated without a mini-program AI value
- **THEN** mini-program AI planning defaults to disabled
