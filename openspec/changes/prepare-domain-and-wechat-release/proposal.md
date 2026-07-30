## Why

The production domain `slls.asia` is now registered and can replace direct IP access, but deployment examples, exposed test ports, mini-program build inputs, and submission guidance are inconsistent. The project needs one repeatable domain-based release path before the first formal WeChat Mini Program submission.

## What Changes

- Make `https://slls.asia` the canonical public origin for H5, admin, backend API access, and the production mini-program build.
- Remove the public IP test-port path from the production Compose configuration and document HTTPS-only public exposure.
- Add deterministic mini-program release validation that rejects an insecure, IP-based, empty, or development-overridable production API origin.
- Add a release command that validates configuration, runs tests and type checks, and produces the WeChat build artifact.
- Consolidate the deployment guide and add an operator checklist covering DNS, TLS, WeChat legal domains, real login, smoke tests, upload, review, and rollback preparation.
- Keep upload, review submission, and final publication as explicit operator actions because they require an authenticated WeChat account and external approval.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deployment-operations`: Define the canonical HTTPS domain, production mini-program build validation, and auditable release readiness checks.

## Impact

- Production configuration: `docker-compose.prod.yml`, environment examples, and Nginx/domain guidance.
- Mini-program build: frontend environment, package scripts, manifest release version, and release checks.
- Operations: deployment documentation, WeChat submission checklist, and CI/release gates.
- No backend API contract or persisted data migration is required.
