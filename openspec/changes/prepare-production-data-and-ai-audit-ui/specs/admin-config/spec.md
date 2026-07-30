## ADDED Requirements

### Requirement: Admin can inspect AI invocations with usable pagination
The system SHALL let administrators inspect AI invocation history in a PC-oriented layout that uses bounded pagination and avoids unnecessary horizontal scrolling on normal admin workspaces.

#### Scenario: View paginated invocation history
- **WHEN** an administrator opens AI model configuration
- **THEN** the console requests invocation history with page and size parameters
- **AND** the console shows the current page, total records, total pages, and page-size control

#### Scenario: Navigate invocation pages
- **WHEN** an administrator clicks previous or next page
- **THEN** the console requests the corresponding page without losing the active filters
- **AND** boundary navigation controls are disabled on the first or last page

#### Scenario: Query invocation filters
- **WHEN** an administrator applies user, job, or status filters
- **THEN** the console resets invocation history to page 1 and refreshes both the history rows and aggregate AI metrics

#### Scenario: Review invocation rows without bottom horizontal scrolling
- **WHEN** an administrator reviews invocation history on a normal PC admin workspace
- **THEN** the invocation panel uses the available workspace width
- **AND** trace, user, job, model, token, and error details wrap within the table instead of requiring a bottom horizontal scrollbar for the default layout
