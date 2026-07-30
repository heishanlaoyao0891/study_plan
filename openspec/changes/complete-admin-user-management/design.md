## Context

`users` already exposes `username`, but the admin directory renders only `nickname`. Registration is intentionally invitation-gated, while user self-service deletion already owns the cascade cleanup and PII anonymization behavior. The management workflow must preserve those two constraints.

## Goals / Non-Goals

**Goals:**

- Make account identity immediately scannable in the PC user directory.
- Give operators a direct, auditable account-creation route with the same credential validation used by registration.
- Allow an authorized administrator to delete a normal user with the same deletion contract used by self-service deletion.
- Record deletion as a sensitive administrator action.

**Non-Goals:**

- Creating administrator accounts or changing roles.
- Restoring erased data or physically removing immutable operational audit metadata.
- Changing the public invitation-based registration flow.

## Decisions

- **Split identity fields in the list.** Render username, nickname, and OpenID in distinct compact columns and search all three. This avoids the current ambiguity where an operator cannot distinguish display identity from password-login identity. A combined label was rejected because it remains hard to copy and scan.
- **Make normal accounts the default view.** Treat a missing or blank status as `active`; query deleted accounts only when explicitly requested. This prevents erased identities from appearing during routine operations while retaining a deliberate recovery and audit view. An unrestricted default was rejected because it makes the destructive action outcome look inconsistent.
- **Create accounts directly from the admin directory.** The administrator supplies a username and nickname; the backend applies the established username/nickname validation, creates the group invite target identity, hashes a cryptographically random initial password, and returns that password only in the create response. This removes unnecessary invitation work while keeping password material out of persistence, lists, and audit logs.
- **Limit login-name changes through an account-event ledger.** Logged-in users can select a WeChat-style 4-24 character login name, including an 11-digit mobile number, at most three times per Shanghai calendar month. Each change creates a `username_change` account event and increments the security version to invalidate other sessions. A verified phone-binding claim is explicitly out of scope because no SMS verification exists.
- **Share destructive-account cleanup.** Extract the existing self-service data erasure helper into a reusable handler-level operation, then call it from a protected admin DELETE endpoint. This preserves one deletion policy rather than allowing two variants to drift.
- **Protect privileged identities.** The admin deletion endpoint rejects both the caller and every admin target. It runs a transaction, anonymizes the user row, invalidates sessions through `security_version`, and adds an admin audit event after completion.

## Risks / Trade-offs

- [Operator deletes the wrong account] → The UI requires a native confirmation that includes username/nickname and the API only targets explicit IDs.
- [Deleted identity could reappear in searches] → The status filter exposes deleted state and removed identity fields remain blank.
- [Cascade misses a future user-owned table] → Reusing the existing cleanup function keeps one testable deletion path; future owned models must be added there.

## Migration Plan

1. Deploy backend endpoint and management UI together.
2. Run focused handler and admin UI tests, then normal backend/admin builds.
3. Roll back application code if needed; no schema or data migration is required.

## Open Questions

- None. Existing invitation registration and account-deletion policies define the required product behavior.
