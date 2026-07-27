# admin-config Delta Specification

## MODIFIED Requirements

### Requirement: Administrator configures AI provider
The system SHALL describe provider configuration as optional asynchronous task decomposition for the always-available backend planning Agent.

#### Scenario: Disable model decomposition
- **WHEN** an administrator disables SiliconFlow or OpenAI-compatible decomposition
- **THEN** smart local planning remains available and no new model jobs are created

#### Scenario: Configure interactive response target
- **WHEN** an administrator changes the local baseline response target
- **THEN** the system accepts a value from 1 to 5 seconds and reports local planning phases that exceed it

#### Scenario: Configure background model budget
- **WHEN** an administrator changes the model job timeout
- **THEN** the system accepts a 5-minute or 10-minute tier, defaults to 5 minutes, and applies it only to newly started jobs

#### Scenario: Test provider configuration
- **WHEN** an administrator runs the provider test
- **THEN** the system validates the structured blueprint contract using the configured background timeout rather than the interactive response target

## ADDED Requirements

### Requirement: Administrator observes planning job health
The system SHALL expose bounded operational metrics for local planning and external model decomposition.

#### Scenario: Review model effectiveness
- **WHEN** an administrator opens AI configuration or operational metrics
- **THEN** the system reports job success and fallback counts, p50/p95 latency, current queue depth, provider/model, token usage where available, and bounded fallback reasons

#### Scenario: Provider latency degrades
- **WHEN** model p95 latency approaches or exceeds the configured background timeout
- **THEN** the admin surface warns that AI decomposition completion is degraded without disabling local planning

#### Scenario: Inspect a failed job
- **WHEN** an administrator reviews recent failed or fallback jobs
- **THEN** the system shows bounded phase and reason metadata without exposing prompts, credentials, or private learning records

#### Scenario: Review Agent repair health
- **WHEN** an administrator reviews AI operations
- **THEN** the system distinguishes provider attempts from successful user generations and reports retry states, truncation, schema repairs, checkpoint resumes, and active Prompt Playbook patterns
