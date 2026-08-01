## 1. Backend Configuration And Enforcement

- [x] 1.1 Add the default-off mini-program AI field to persisted AI configuration and admin request/response contracts.
- [x] 1.2 Add an authenticated fail-closed client-feature endpoint that exposes only mini-program AI availability.
- [x] 1.3 Enforce the switch on explicitly identified mini-program AI plan-job endpoints while preserving H5 and legacy clients.
- [x] 1.4 Record old and new mini-program AI states in the admin audit log.

## 2. Admin And Client Experience

- [x] 2.1 Add the independent mini-program AI switch and platform-risk warning to the PC admin AI configuration page.
- [x] 2.2 Add compile-time platform metadata to the shared UniApp request layer.
- [x] 2.3 Make the mini-program plan page fail closed, hide all AI UI/polling when disabled, and promote manual creation.
- [x] 2.4 Guard direct mini-program AI-page access and submission using the same feature response without changing H5.

## 3. Verification And Documentation

- [x] 3.1 Add backend tests for defaults, feature privacy, channel enforcement, H5/legacy compatibility, and audit logging.
- [x] 3.2 Add admin and frontend contract tests for the switch, warning, channel headers, hidden state, polling suppression, and direct-route guard.
- [x] 3.3 Run backend tests and vet, admin tests/type-check/build, frontend tests/type-check/H5/mini-program builds, and strict OpenSpec validation.
- [x] 3.4 Update the living project scan with the new active change, behavior, validation, and incremental log.
