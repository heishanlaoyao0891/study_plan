## MODIFIED Requirements

### Requirement: User can view efficiency statistics
The system SHALL provide study analytics for the last 7 days, last 30 days, and last 12 calendar months across time and plan dimensions.

#### Scenario: User selects a time dimension
- **WHEN** the user selects a supported period and time dimension
- **THEN** the system returns zero-filled day or month buckets with study, planned, overtime, task, and completion metrics

#### Scenario: User selects a plan dimension
- **WHEN** the user selects a supported period and plan dimension
- **THEN** the system groups the same metrics by plan and orders plans by recorded study minutes

#### Scenario: User opens statistics on H5 or mini-program
- **WHEN** trend data is available
- **THEN** the client renders selectable native bar charts without requiring a platform-specific chart library
