# ai-plan-generator Specification

## Purpose
TBD - created by archiving change study-checkin-miniapp. Update Purpose after archive.
## Requirements
### Requirement: AI generates a study plan from user description
The system SHALL accept a natural language learning goal and generate a structured daily task plan via AI.

#### Scenario: Generate a plan
- **WHEN** user sends POST /api/ai/generate-plan with goal, hours_per_day, start_date, available_time_slot, and optional skip_dates
- **THEN** system calls AI API and returns an array of DailyTask objects with date, title, description, scheduled_start, scheduled_end

### Requirement: AI asks for user availability before generating
The system SHALL collect user's daily available hours and time slots before AI generation.

#### Scenario: Provide availability
- **WHEN** user starts AI plan generation
- **THEN** system prompts user for: daily available hours, preferred time slot (e.g. 20:00-22:00), start date, and dates to skip

### Requirement: AI estimates total duration based on goal
The system SHALL estimate the total study duration for the goal and distribute it across days.

#### Scenario: Estimate and distribute
- **WHEN** AI receives a goal like "learn Go"
- **THEN** AI estimates total hours needed (e.g. 60h) and generates daily tasks spread across days based on daily available hours

### Requirement: AI considers historical learning ability
The system SHALL optionally use user's historical study data to adjust plan difficulty and pace.

#### Scenario: Personalized pace
- **WHEN** AI generates plan and user has historical data
- **THEN** AI adjusts daily task density based on user's past completion rate

### Requirement: User can edit AI-generated plan
The system SHALL allow users to modify, add, delete, or reorder tasks after AI generation.

#### Scenario: Edit generated tasks
- **WHEN** user edits any task in the generated plan
- **THEN** system updates the task immediately

#### Scenario: Regenerate a plan
- **WHEN** user is unsatisfied with AI output and requests regeneration
- **THEN** system calls AI again with refined parameters

### Requirement: Plan creation respects user's availability
The system SHALL skip dates the user marks as unavailable when generating tasks.

#### Scenario: Skip unavailable dates
- **WHEN** user marks certain dates as unavailable
- **THEN** AI does not create tasks on those dates

