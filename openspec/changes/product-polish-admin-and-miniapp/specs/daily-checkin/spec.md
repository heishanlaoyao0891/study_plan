## MODIFIED Requirements

### Requirement: Checkin requires all tasks done
The system SHALL treat task completion as the source of truth for completing the related daily check-in.

#### Scenario: Complete task from mini program
- **WHEN** the user completes the final remaining task for a plan on a date
- **THEN** the backend completes the daily check-in for that plan/date
- **AND** the client does not perform a duplicate check-in mutation for the same completion

### Requirement: Completed checkin earns slack time
The system SHALL award slack minutes once per completed daily check-in.

#### Scenario: Task completion triggers check-in reward
- **WHEN** task completion causes the daily check-in to become complete
- **THEN** the system awards configured slack minutes once
- **AND** repeated client refreshes or duplicate UI actions do not award duplicate slack time

## ADDED Requirements

### Requirement: Mini program check-in experience is delightful
The system SHALL make the daily check-in page feel warm, cute, and rewarding while preserving clear task controls.

#### Scenario: User opens today page
- **WHEN** the user opens the check-in tab
- **THEN** the page shows a cute first impression, clear progress, friendly empty states, and distinctive primary action buttons
- **AND** the user can still quickly start, stop, and complete study tasks
