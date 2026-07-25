# admin-config Delta Specification

## MODIFIED Requirements

### Requirement: Administrator configures AI provider
The system SHALL describe provider configuration as optional model enrichment for the always-available backend planning Agent.

#### Scenario: Disable model enrichment
- **WHEN** an administrator disables SiliconFlow enrichment
- **THEN** smart local planning remains available and only external enrichment stops

#### Scenario: Configure interactive budget
- **WHEN** an administrator sets the enrichment timeout budget
- **THEN** the system validates a bounded value that cannot exceed the overall planning request budget
