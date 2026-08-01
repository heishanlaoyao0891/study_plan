## ADDED Requirements

### Requirement: Mini-program AI planning follows runtime availability
The system SHALL condition the supported mini-program AI planning UI and plan-job APIs on the persisted mini-program AI setting while leaving H5 AI planning unchanged.

#### Scenario: Disabled mini-program opens plans
- **WHEN** a supported mini-program opens the plan page while mini-program AI is disabled
- **THEN** it shows manual plan creation without an AI hero, AI job state, AI polling, or AI navigation

#### Scenario: Disabled mini-program opens AI route directly
- **WHEN** a supported mini-program navigates directly to the AI planning page while mini-program AI is disabled
- **THEN** the client returns to plans and does not submit or poll an AI job

#### Scenario: Disabled mini-program calls an AI job endpoint
- **WHEN** a request explicitly identified as `mp-weixin` reads or submits an AI plan job while mini-program AI is disabled
- **THEN** the backend rejects the request with a clear feature-disabled response

#### Scenario: Enabled mini-program uses AI planning
- **WHEN** mini-program AI is enabled and a supported mini-program opens plans
- **THEN** the AI planning entry, durable job status, navigation, and generation workflow are available

#### Scenario: H5 uses AI while mini-program AI is disabled
- **WHEN** an H5 client uses AI planning while mini-program AI is disabled and the provider is enabled
- **THEN** the normal H5 AI planning workflow remains available

#### Scenario: Legacy published mini-program sends no channel metadata
- **WHEN** version 1.0.1 calls AI planning without supported client platform metadata
- **THEN** the backend preserves its existing behavior for backward compatibility
