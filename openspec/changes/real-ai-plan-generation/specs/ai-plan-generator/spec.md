## MODIFIED Requirements

### Requirement: AI generates a study plan from user description
The system SHALL accept a natural language learning goal and generate a structured daily task plan through a configured AI planning agent, validating the model output before it can be persisted.

#### Scenario: Generate a validated plan preview
- **WHEN** user sends a generation request with goal, availability, start date, and skip dates
- **THEN** system builds a business-specific planning context, calls the configured AI provider, and returns a validated editable preview

#### Scenario: Invalid AI output rejected
- **WHEN** AI returns malformed JSON, missing required fields, or tasks on skipped dates
- **THEN** system rejects the output with a clear error and does not persist tasks

### Requirement: User can edit AI-generated plan
The system SHALL allow users to modify, add, delete, or reorder tasks in an AI-generated preview before committing it.

#### Scenario: Commit edited preview
- **WHEN** user accepts an edited AI preview
- **THEN** system creates the plan and tasks exactly from the accepted preview

#### Scenario: Regenerate with refinements
- **WHEN** user requests regeneration with additional instructions
- **THEN** system calls AI again using the original inputs plus refinements

## ADDED Requirements

### Requirement: AI usage is controlled
The system SHALL enforce limits for AI generation requests to prevent accidental cost or abuse.

#### Scenario: User exceeds generation limit
- **WHEN** user exceeds the configured generation limit
- **THEN** system rejects new generation requests with a rate-limit message

#### Scenario: Default daily generation limit
- **WHEN** no custom AI usage limit is configured
- **THEN** system applies a default limit of 5 generation requests per user per day

### Requirement: AI provider is configurable
The system SHALL allow administrators to configure and validate the AI provider used by plan generation.

#### Scenario: Configure OpenAI-compatible provider
- **WHEN** admin configures base URL, model name, API key, timeout, and enabled state
- **THEN** system uses that provider for future AI generation requests

#### Scenario: Test provider connectivity
- **WHEN** admin runs an AI provider test from the PC admin console
- **THEN** system sends a minimal test request and reports whether provider access and response parsing succeeded

### Requirement: AI planning uses historical learning data
The system SHALL use relevant user history to adjust generated plan difficulty and pacing.

#### Scenario: User has low recent completion rate
- **WHEN** AI generates a plan for a user with low recent completion rate
- **THEN** system reduces daily load or increases schedule buffer in the generated preview

#### Scenario: User has strong recent completion rate
- **WHEN** AI generates a plan for a user with strong recent completion rate
- **THEN** system may generate a denser plan while staying within configured workload limits

### Requirement: AI agent can use controlled planning tools
The system SHALL allow the planning agent to use backend-owned allowlisted tools for user-specific learning context and schedule checks.

#### Scenario: Agent retrieves learning profile
- **WHEN** plan generation needs historical user context
- **THEN** agent may call an allowlisted backend tool that returns aggregated completion rate, average study minutes, streak, and postpone frequency for the authenticated user

#### Scenario: Agent checks schedule conflicts
- **WHEN** generated tasks include planned dates and times
- **THEN** agent may call an allowlisted backend tool to detect conflicts with the user's existing active plan load

#### Scenario: Tool access is restricted
- **WHEN** the model requests data outside the allowlisted planning tools
- **THEN** system refuses the tool request and does not expose raw database, credentials, SQL, or cross-user data

### Requirement: AI generation has fallback behavior
The system SHALL provide safe fallback behavior when the model provider fails or returns unusable output.

#### Scenario: Provider output is invalid but repairable
- **WHEN** model output has minor JSON formatting issues
- **THEN** system attempts safe repair and validates the repaired output before returning a preview

#### Scenario: Provider unavailable
- **WHEN** provider times out or rejects the request
- **THEN** system retries according to policy and may return a deterministic fallback preview or a clear unavailable error
