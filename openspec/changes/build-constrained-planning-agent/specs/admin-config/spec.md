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

#### Scenario: Aggregate provider Token usage
- **WHEN** an administrator reviews prompt, completion, or total Token usage
- **THEN** the system sums the recent immutable invocation traces rather than legacy orchestration-job Token fields, including all recorded provider and repair attempts in the metrics window

#### Scenario: Refresh invocation operations
- **WHEN** an administrator refreshes recent AI invocation history
- **THEN** the administration page reloads aggregate planning metrics in the same action so Token totals reflect newly completed calls

### Requirement: Administrator inspects every AI invocation
The system SHALL expose a paginated, filterable audit ledger containing one immutable trace for every external provider HTTP attempt.

#### Scenario: Inspect successful and failed attempts
- **WHEN** an administrator opens recent AI invocation traces
- **THEN** each row identifies trace ID, user ID when applicable, job type/ID, Agent phase, batch and repair attempt, provider retry, provider/model, status, timing, HTTP/finish metadata, token usage, and bounded failure reason

#### Scenario: Filter invocation history
- **WHEN** an administrator filters by user, job, status, provider, or time range
- **THEN** the system returns only matching traces with bounded pagination and newest attempts first

#### Scenario: Protect sensitive AI content
- **WHEN** invocation history is stored or displayed
- **THEN** it contains request fingerprints and sizes but never API keys, authorization headers, raw prompts, raw responses, private learning records, or unbounded provider errors
