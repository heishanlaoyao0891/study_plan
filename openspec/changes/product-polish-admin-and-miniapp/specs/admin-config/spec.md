## MODIFIED Requirements

### Requirement: Admin can view user list
The system SHALL allow administrators to view registered users in a Chinese, operator-friendly admin console.

#### Scenario: View users with Chinese admin UI
- **WHEN** an administrator opens the user-management page
- **THEN** the page shows Chinese navigation, filters, table headers, status labels, and action labels
- **AND** the visual layout supports quick scanning on PC screens

### Requirement: Admin authentication
The system SHALL restrict admin endpoints to authorized users only and provide a clear recovery path when an admin session expires.

#### Scenario: Admin session expired
- **WHEN** an admin API request returns unauthorized or forbidden because the session is invalid
- **THEN** the admin console clears the stale session and guides the admin back to the login page

## ADDED Requirements

### Requirement: Admin console has polished Chinese information architecture
The system SHALL present the PC admin console with Chinese-first navigation, page titles, primary actions, table labels, and form labels.

#### Scenario: Operator scans admin dashboard
- **WHEN** an operator opens the PC admin console
- **THEN** the sidebar, topbar, overview metrics, and page headings use Chinese product terminology
- **AND** the console uses a consistent dashboard style across pages

### Requirement: Admin console preserves operational density without looking unfinished
The system SHALL keep admin pages dense enough for operations while using consistent spacing, cards, buttons, forms, and tables.

#### Scenario: Operator reviews tabular data
- **WHEN** an operator views users, audit logs, suspicious records, feedback, or configuration pages
- **THEN** tables and panels use consistent typography, status treatments, and action affordances
