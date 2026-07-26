# admin-config Delta Specification

## MODIFIED Requirements

### Requirement: Admin can ban/unban users
The system SHALL store permanent bans using a server-owned sentinel and SHALL expose permanence explicitly in user-facing ban responses.

#### Scenario: Permanent ban is enforced
- **WHEN** an administrator creates a permanent ban
- **THEN** the server stores its canonical sentinel and subsequent ban responses set `permanent=true`

#### Scenario: Timed ban is enforced
- **WHEN** an administrator creates a positive-duration ban
- **THEN** subsequent active-ban responses set `permanent=false` and return the exact RFC3339 deadline

#### Scenario: Timed ban expires
- **WHEN** authentication evaluates a timed ban at or after its deadline
- **THEN** the system clears `banned_until` and `banned_reason` and permits normal authentication
