# notification Delta Specification

## MODIFIED Requirements

### Requirement: User can subscribe to reminders
The system SHALL request actual WeChat template authorization in the mini program and SHALL not offer WeChat subscription controls in H5.

#### Scenario: Mini program authorizes templates
- **WHEN** the user taps subscribe and accepts one or more templates in `requestSubscribeMessage`
- **THEN** the client submits per-template results and the backend records only accepted subscriptions

#### Scenario: H5 opens reminder settings
- **WHEN** an H5 user opens reminder settings
- **THEN** the system explains that WeChat subscription messages require the linked mini program and does not create subscriptions

### Requirement: Due reminders are delivered idempotently
The system SHALL run scheduled due-event processing and create at most one delivery per logical reminder event.

#### Scenario: Worker finds due event
- **WHEN** an enabled configured subscribed reminder becomes due
- **THEN** the worker claims a unique event, sends the configured template payload, and records the provider result

#### Scenario: Worker retries same event
- **WHEN** another worker cycle observes an already claimed or sent event
- **THEN** it does not send a duplicate message

### Requirement: Group nudge uses subscription delivery
The system SHALL send a configured group-nudge template when the target has authorized it and otherwise record the skipped reason.

#### Scenario: Deliver group nudge
- **WHEN** a member nudges another member who authorized the enabled group-nudge template
- **THEN** the system sends one WeChat subscription message and records the result
