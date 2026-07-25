# admin-config Delta Specification

## ADDED Requirements

### Requirement: Administrator operates password recovery
The system SHALL let administrators generate one-time 30-minute password reset codes without exposing password hashes.

#### Scenario: Generate reset code
- **WHEN** an administrator requests recovery for an active user
- **THEN** the system returns the plaintext reset code once and stores only its hash and audit metadata

### Requirement: Administrator configures complete reminder templates
The system SHALL let administrators configure each reminder's template ID, enable state, page target, and template-field mapping, including group nudges.

#### Scenario: Invalid template configuration
- **WHEN** required template fields or mapping are missing
- **THEN** the system refuses to enable that reminder and explains the missing configuration
