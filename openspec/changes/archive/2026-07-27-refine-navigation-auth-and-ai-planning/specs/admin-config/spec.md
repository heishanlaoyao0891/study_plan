# admin-config Delta Specification

## ADDED Requirements

### Requirement: Admin navigation keeps stable dimensions
The PC admin console SHALL keep its sidebar at a stable compact width and SHALL constrain route content to the workspace without allowing overview metrics, charts, tables, loading states, or errors to resize the navigation column.

#### Scenario: Operator opens overview
- **WHEN** an administrator selects `运营总览`
- **THEN** the sidebar width and navigation label layout remain unchanged from other admin routes

#### Scenario: Workspace contains wide content
- **WHEN** an admin route renders content wider than the available workspace
- **THEN** the content wraps, shrinks, or scrolls inside the workspace without widening the sidebar

#### Scenario: Admin viewport narrows
- **WHEN** the PC console is viewed at its supported minimum desktop width
- **THEN** navigation remains usable and workspace content does not overlap or displace it
