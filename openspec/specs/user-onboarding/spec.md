# user-onboarding Specification

## Purpose
TBD - created by archiving change onboarding-and-recovery. Update Purpose after archive.
## Requirements
### Requirement: New user receives guided first-run onboarding
The system SHALL guide new users through the minimum steps needed to begin studying.

#### Scenario: New user starts app
- **WHEN** a new user opens the mini program after login
- **THEN** system guides the user through phone binding, optional reminder setup, first plan creation or AI generation, and today's first task

### Requirement: Reminder subscription prompts are contextual
The system SHALL request reminder subscriptions only in relevant contexts and avoid repeated prompts after refusal.

#### Scenario: User refuses reminder subscription
- **WHEN** user declines a reminder subscription prompt
- **THEN** system does not repeatedly prompt immediately and leaves a retry entry in reminder settings

### Requirement: User can recover from missed schedule
The system SHALL offer schedule recovery when the user falls behind.

#### Scenario: User has overdue tasks
- **WHEN** user has missed days, overdue tasks, or pending decisions
- **THEN** system offers a recovery flow that previews revised future tasks before applying changes

#### Scenario: AI unavailable for recovery
- **WHEN** AI recovery generation is unavailable
- **THEN** system provides a rule-based recovery preview instead

