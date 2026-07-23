## 1. Backend API Tests

- [x] 1.1 Add auth/login tests for mock and blocked users
- [x] 1.2 Add plan ownership and overload tests
- [x] 1.3 Add task lifecycle tests
- [x] 1.4 Add check-in reward tests
- [x] 1.5 Add admin authorization tests
- [x] 1.6 Add focused scheduling edge-case tests for midnight decision, makeup, postpone, and auto check-in

## 2. Database Reliability

- [x] 2.1 Review and add missing indexes/unique constraints
- [x] 2.2 Add migration verification test
- [x] 2.3 Document backup and restore verification

## 3. Frontend Quality

- [x] 3.1 Add type-check command to verification flow
- [x] 3.2 Add critical flow checklist for mini program preview
- [x] 3.3 Fix issues discovered by type-check/build
- [x] 3.4 Add admin console type-check/build to verification flow when admin project exists

## 4. Verification Script

- [x] 4.1 Add PowerShell script for backend tests, backend build, frontend type-check/build, admin type-check/build if present, and OpenSpec validation
- [x] 4.2 Document release gates
- [x] 4.3 Keep GitHub Actions optional for this iteration

## 5. Verification

- [x] 5.1 Run backend tests
- [x] 5.2 Run backend build
- [x] 5.3 Run frontend build
- [x] 5.4 Run frontend type-check if supported
- [x] 5.5 Run admin console build/type-check if admin project exists
- [x] 5.6 Run OpenSpec validation for active changes
