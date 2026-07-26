## MODIFIED Requirements

### Requirement: System controls planned schedule overlap and overload
The system SHALL require every task to have less than 60 distinct covered minutes and SHALL additionally limit a task fully contained by another task to at most 30 covered minutes.

#### Scenario: Contained task lasts 30 minutes
- **WHEN** task A is 30 minutes and its full interval is contained by task B
- **THEN** the schedule remains valid if no other overlap rule is violated

#### Scenario: Contained task exceeds 30 minutes
- **WHEN** task A lasts more than 30 minutes and its full interval is contained by task B
- **THEN** the schedule is rejected and identifies A as fully contained
