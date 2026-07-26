## MODIFIED Requirements

### Requirement: AI generates a study plan from user description
The system SHALL preserve at least the first requested available hour in each generated task and MAY adapt only workload above that hour using historical learning signals.

#### Scenario: User requests one available hour
- **WHEN** the user requests one hour and the available slot contains at least 60 minutes
- **THEN** each generated task uses 60 minutes regardless of a low historical completion rate or shorter average session

#### Scenario: Future work exists
- **WHEN** future scheduled tasks have not yet been completed
- **THEN** those future dates do not reduce the historical completion rate used for planning
