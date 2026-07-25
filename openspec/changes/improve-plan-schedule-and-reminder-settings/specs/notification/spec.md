# notification Delta Specification

## MODIFIED Requirements

### Requirement: User can subscribe to reminders
The system SHALL present every enabled WeChat reminder template with its purpose and current persisted authorization record, and SHALL request authorization for one template per direct mini-program user tap.

#### Scenario: Mini-program user views enabled reminders
- **WHEN** reminder settings loads in the WeChat mini program
- **THEN** the client combines enabled template metadata with the user's subscription records
- **AND** shows a user-friendly name, purpose, and saved authorization state for each enabled reminder type

#### Scenario: User authorizes one reminder
- **WHEN** the user directly taps authorize on one reminder card
- **THEN** the client invokes `requestSubscribeMessage` with only that card's template ID
- **AND** submits that reminder type, template ID, and per-template result to the subscription API
- **AND** the server persists it only when both identifiers match the current enabled template

#### Scenario: Authorization completes
- **WHEN** WeChat accepts, rejects, or closes one authorization request
- **THEN** the client refreshes enabled metadata and persisted subscription records
- **AND** reports the result without changing other reminder records

#### Scenario: User understands one-time consumption
- **WHEN** the user views a saved authorization record
- **THEN** the interface explains that WeChat generally consumes one authorization per delivered message
- **AND** provides a direct re-authorize action for another reminder opportunity

#### Scenario: User cancels all reminders
- **WHEN** the user confirms cancel-all
- **THEN** all persisted reminder subscriptions are removed using the existing API
- **AND** the displayed authorization records refresh

#### Scenario: H5 opens reminder settings
- **WHEN** an H5 user opens reminder settings
- **THEN** the system explains that authorization must be completed in the linked WeChat mini program
- **AND** does not render or invoke WeChat authorization controls
