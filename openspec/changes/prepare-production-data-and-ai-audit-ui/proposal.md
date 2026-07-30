## Why

The mini program is entering release preparation, so production must be cleaned of unambiguous test accounts and their dependent records with a recoverable backup. The administrator AI invocation history also needs to be usable on PC screens: the current narrow panel hides important fields behind a bottom horizontal scrollbar and does not expose the backend pagination already available.

## What Changes

- Perform a production data cleanup run that inventories accounts first, backs up the SQLite database, and removes only clearly identified test users through the same full user-data cleanup policy used by account deletion.
- Make the AI invocation history panel use the available admin workspace width.
- Add visible pagination controls, total count, page-size selection, and boundary states to the AI invocation history table.
- Keep invocation refresh behavior coupled with aggregate AI token metrics refresh so operators see current ledger totals after reviewing recent traces.
- Add an invitation-help entry on the authentication home page that opens the administrator's QR code for users who need to request an invitation.
- Add focused admin layout/contract tests for the wider invocation panel and pagination controls.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `admin-config`: AI invocation inspection must be operationally usable with bounded pagination and a layout that does not require bottom horizontal scrolling on normal PC admin workspaces.
- `user-auth`: WeChat existing-account linking must tolerate legacy incomplete OpenID holder accounts, return clear localized conflicts for already-linked identities, and expose administrator QR contact help for users without invitations.

## Impact

- Affected code: `admin/src/views/AIConfigView.vue`, `admin/src/api.ts` if needed, `admin/tests/layout.test.mjs`, `backend/handlers/h5_auth.go`, `backend/handlers/h5_auth_test.go`, `frontend/src/pages/login/login.vue`, `frontend/tests/navigation-auth.test.mjs`.
- Affected operations: production SQLite backup and deletion of confirmed test-user data.
- Affected docs/status: `docs/project-scan.html` must record the new active change, validation, and production cleanup result.
