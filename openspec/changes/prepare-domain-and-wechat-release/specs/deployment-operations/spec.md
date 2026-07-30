# deployment-operations Delta Specification

## MODIFIED Requirements

### Requirement: Tencent Cloud deployment is documented
The system SHALL document a production deployment path for Tencent Cloud using the registered HTTPS domain while preserving local verification and rollback steps.

#### Scenario: Deploy to Tencent Cloud
- **WHEN** an operator follows the production deployment guide
- **THEN** backend, H5, and admin run behind `https://slls.asia`, the WeChat legal request domain is configured, and direct public IP test access is not required

#### Scenario: Roll back a failed application release
- **WHEN** post-deployment smoke checks fail
- **THEN** the guide identifies how to restore the previous application revision without replacing persistent SQLite or object-storage volumes

## ADDED Requirements

### Requirement: Production uses a canonical HTTPS origin
The system SHALL use `https://slls.asia` as the canonical public origin for H5, admin, backend API requests, and production mini-program requests.

#### Scenario: Access a production surface
- **WHEN** a user opens H5, an administrator opens `/admin/`, or the mini program calls `/api/`
- **THEN** the request uses the registered HTTPS domain with a valid certificate instead of a raw IP address or public test port

#### Scenario: Build with an unsafe API origin
- **WHEN** the production mini-program API origin is empty, non-HTTPS, an IP literal, contains an unexpected path, or enables a development override
- **THEN** the release validation fails before producing a submission artifact

### Requirement: Mini-program release is reproducible
The project SHALL provide one production release command that validates manifest and environment metadata, runs automated quality checks, and generates the WeChat artifact.

#### Scenario: Build a valid release candidate
- **WHEN** the canonical origin, real AppID, disabled development override, and incremented version metadata are present
- **THEN** the command runs tests and type checking, builds `frontend/dist/build/mp-weixin`, and verifies the generated artifact contains the expected origin and AppID

#### Scenario: Hand off for WeChat publication
- **WHEN** the local release candidate passes
- **THEN** the operator checklist identifies legal-domain configuration, real-device smoke tests, upload version notes, privacy/category review, submission, publication, and rollback evidence
