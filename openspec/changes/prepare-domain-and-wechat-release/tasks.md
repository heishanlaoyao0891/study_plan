## 1. Canonical Production Domain

- [x] 1.1 Remove the public IP test port from production Compose and environment examples.
- [x] 1.2 Align Nginx and deployment documentation on `https://slls.asia` for H5, admin, health, and API access.
- [x] 1.3 Document DNS, TLS, loopback diagnostics, and rollback behavior without bypassing the canonical origin.

## 2. Mini-Program Release Candidate

- [x] 2.1 Increment mini-program version metadata and retain the registered WeChat AppID.
- [x] 2.2 Add a deterministic release validator for HTTPS domain, AppID, development override, version, and generated artifact contents.
- [x] 2.3 Add a package release command that runs frontend tests, type checking, production build, and artifact validation.

## 3. Operator Handoff

- [x] 3.1 Add a WeChat submission checklist covering legal domains, privacy/category settings, real-device smoke tests, upload, review, publication, and evidence.
- [x] 3.2 Update release gates and CI/CD documentation to use the canonical domain release path.

## 4. Verification and Delivery

- [x] 4.1 Run frontend tests, type checking, mini-program release build, Compose YAML validation, Nginx/static checks, and strict OpenSpec validation.
- [x] 4.2 Update the living project scan with the new active change, release posture, validation results, and remaining external publication steps.
- [x] 4.3 Commit and push the completed release-preparation change.
