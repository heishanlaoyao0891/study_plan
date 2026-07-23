# deployment-operations Specification

## Purpose
TBD - created by archiving change production-hardening-study-checkin. Update Purpose after archive.
## Requirements
### Requirement: Production startup validates required configuration
The system SHALL fail startup in production when required secrets or runtime settings are missing.

#### Scenario: Missing required production secret
- **WHEN** backend starts with production settings and `JWT_SECRET` is empty
- **THEN** startup fails with a clear configuration error

### Requirement: Backend emits structured operational logs
The system SHALL write structured logs for requests, important user actions, and backend errors without leaking secrets.

#### Scenario: API request logged
- **WHEN** an authenticated API request completes
- **THEN** logs include route, status, latency, and user id without logging the JWT value

### Requirement: SQLite database can be backed up
The system SHALL provide a documented backup process for the SQLite database used in small-scale deployments.

#### Scenario: Backup command executed
- **WHEN** operator runs the backup command
- **THEN** a timestamped database backup is created without corrupting the active database

### Requirement: SQLite data can be archived to MySQL when configured
The system SHALL optionally archive selected SQLite data to MySQL while keeping SQLite as the primary runtime database.

#### Scenario: Archive sync enabled
- **WHEN** archive sync is enabled and MySQL DSN is configured
- **THEN** system periodically copies selected SQLite tables to MySQL for backup, reporting, or future migration readiness

#### Scenario: Archive sync disabled
- **WHEN** archive sync is disabled
- **THEN** system does not require MySQL and normal app behavior continues using SQLite only

#### Scenario: Archive sync fails
- **WHEN** MySQL archival fails
- **THEN** system logs the failure and keeps normal study/check-in operations available

### Requirement: Tencent Cloud deployment is documented
The system SHALL document production deployment steps for Tencent Cloud while preserving local verification steps.

#### Scenario: Deploy to Tencent Cloud
- **WHEN** operator follows the production deployment guide
- **THEN** backend runs on Tencent Cloud with HTTPS reverse proxy and WeChat legal request domain configuration

