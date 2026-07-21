## ADDED Requirements

### Requirement: Makeup study time consumes slack minutes
The system SHALL consume slack minutes when users manually make up study time, using a configurable ratio.

#### Scenario: Makeup accepted with sufficient slack
- **WHEN** user submits makeup study time and has sufficient slack balance
- **THEN** system deducts slack minutes according to the configured ratio and records the deduction

#### Scenario: Makeup rejected with insufficient slack
- **WHEN** user submits makeup study time but lacks required slack balance
- **THEN** system rejects the makeup request under MVP policy

### Requirement: Suspicious study records are flagged
The system SHALL mark abnormal study records for admin review without automatically banning users.

#### Scenario: Excessive makeup duration
- **WHEN** user submits an unusually long makeup or study record
- **THEN** system marks the record suspicious for admin visibility

### Requirement: Slack minutes have no monetary value
The system SHALL present slack minutes only as rest/activity balance and not as money or redeemable value.

#### Scenario: User views slack page
- **WHEN** user opens slack page
- **THEN** system describes slack minutes as rest/activity balance without money-related wording

### Requirement: User receives rest balance suggestions
The system SHALL provide simple study/rest balance suggestions based on study and slack history.

#### Scenario: High study week
- **WHEN** user has high study minutes and available slack balance
- **THEN** system may suggest a reasonable rest activity or rest duration
