## 1. Admin AI invocation history

- [x] 1.1 Add a failing admin layout/contract test for invocation pagination state and full-width layout.
- [x] 1.2 Update `AIConfigView.vue` to keep `items`, `total`, `page`, and `size` in state and query the backend with the selected page.
- [x] 1.3 Add visible total/page/page-size controls with previous/next boundary states and filter reset behavior.
- [x] 1.4 Make the invocation panel use the available workspace width and remove the table layout that forces a bottom horizontal scrollbar.
- [x] 1.5 Validate admin tests, type-check, and production build.

## 2. WeChat existing-account linking

- [x] 2.1 Add backend regression tests for legacy incomplete OpenID holder migration and localized identity conflict.
- [x] 2.2 Update `WeChatLink` to migrate only incomplete legacy OpenID holders and keep complete holder conflicts protected.
- [x] 2.3 Validate the focused WeChat/H5 authentication test loop.
- [x] 2.4 Add the authentication-page administrator QR entry and frontend contract coverage.

## 3. Production release cleanup

- [x] 3.1 Inventory production users and dependent data with read-only queries.
- [x] 3.2 Create a timestamped production SQLite backup before deletion.
- [x] 3.3 Delete only unambiguous test users and dependent data in a verified transaction.
- [x] 3.4 Report exact deleted users/data counts and the backup location.

## 4. Documentation and verification

- [x] 4.1 Validate the OpenSpec change strictly.
- [x] 4.2 Update `docs/project-scan.html` with the new change, task status, validation, and cleanup result.
