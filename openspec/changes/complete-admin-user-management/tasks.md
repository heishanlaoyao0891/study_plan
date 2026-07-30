## 1. User directory API

- [x] 1.1 Extend user-directory search to match login usernames as well as nickname and OpenID.
- [x] 1.2 Add an administrator-only user deletion endpoint that rejects protected administrator accounts and records an audit event.
- [x] 1.3 Reuse the established account data deletion and anonymization policy for administrator-initiated deletion.

## 2. Admin console workflow

- [x] 2.1 Display username, nickname, and OpenID as separate user-directory columns and update search copy.
- [x] 2.2 Add a directory action that leads administrators to invitation-based account creation.
- [x] 2.3 Add a confirmed delete action with loading and error states for eligible users.

## 3. Verification and project index

- [x] 3.1 Add backend regression coverage for username search, deletion, privilege protection, and audit logging.
- [x] 3.2 Add admin UI contract coverage for identity columns and account-management actions.
- [x] 3.3 Run focused tests and builds, validate the OpenSpec change, and update the living project scan report.

## 4. User directory state visibility refinement

- [x] 4.1 Make normal, unbanned accounts the default API and console result, excluding deleted and inactive accounts.
- [x] 4.2 Add all, normal, banned, and deleted filters with backend and admin UI regression coverage.

## 5. Direct account creation and login-name control

- [x] 5.1 Replace the invitation redirect with an in-directory administrator account-creation form and one-time generated initial password.
- [x] 5.2 Add direct account creation API validation, hashing, group invite-target identity generation, and administrator audit logging.
- [x] 5.3 Let users update login names through account settings, record each change, refresh sessions, and enforce a three-per-Shanghai-calendar-month limit.
- [x] 5.4 Run cross-surface regression tests, strict validation, and update the living project scan report.
