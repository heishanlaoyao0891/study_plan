## MODIFIED Requirements

### Requirement: AI generates a study plan from user description
The system SHALL accept a natural language learning goal and generate a validated structured daily task plan whose tasks include concrete execution objectives.

#### Scenario: Generate a plan
- **WHEN** user sends POST /api/ai/generate-plan with goal, hours_per_day, start_date, available_time_slot, and optional skip_dates
- **THEN** system calls AI API and returns DailyTask previews with date, title, objective, description, scheduled_start, and scheduled_end

#### Scenario: Generate a validated plan preview
- **WHEN** user sends a generation request with goal, availability, start date, and skip dates
- **THEN** system builds a business-specific planning context, calls the configured AI provider, and returns a validated editable preview with an objective for every task

#### Scenario: Invalid AI output rejected
- **WHEN** AI returns malformed JSON, missing required fields, missing task objectives, objectives that only repeat titles, or tasks on skipped dates
- **THEN** system rejects the output with a clear error and does not persist tasks
