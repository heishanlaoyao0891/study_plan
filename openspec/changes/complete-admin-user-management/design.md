## Context

`users` already exposes `username`, but the admin directory renders only `nickname`. Registration is intentionally invitation-gated, while user self-service deletion already owns the cascade cleanup and PII anonymization behavior. The management workflow must preserve those two constraints.

## Goals / Non-Goals

**Goals:**

- Make account identity immediately scannable in the PC user directory.
- Give operators an account-creation route without bypassing invitation and credential validation.
- Allow an authorized administrator to delete a normal user with the same deletion contract used by self-service deletion.
- Record deletion as a sensitive administrator action.

**Non-Goals:**

- Creating administrator accounts or changing roles.
- Restoring erased data or physically removing immutable operational audit metadata.
- Replacing invitation-based public registration with direct account provisioning.

## Decisions

- **Split identity fields in the list.** Render username, nickname, and OpenID in distinct compact columns and search all three. This avoids the current ambiguity where an operator cannot distinguish display identity from password-login identity. A combined label was rejected because it remains hard to copy and scan.
- **Use registration invitations as the add-user mechanism.** The directory's add action routes to the existing invitation page. Direct administrator creation was rejected because it duplicates password hashing, uniqueness, and invitation policy while creating unclear first-login semantics.
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
