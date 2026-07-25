## Context

The existing `users` row is both the account and the WeChat identity. H5 requires an account credential that does not depend on OpenID. Public self-registration is intentionally out of scope; administrators invite known early users.

## Goals / Non-Goals

**Goals:**

- Make one user ID authoritative across H5 and WeChat.
- Avoid external SMS or email provider requirements.
- Prevent uninvited or unlinked WeChat identities from accessing study data.
- Give administrators auditable control over early account creation.

**Non-Goals:**

- Phone or email collection and verification.
- Self-service password recovery.
- Public registration without an invitation.
- Automatic merging of two existing accounts.

## Decisions

### One row represents the shared account

`users.username_normalized` is unique when present, `users.open_id` is unique when present, and `users.password_hash` stores a bcrypt hash. Plans, tasks, check-ins, balances, and groups continue referencing the same user ID regardless of login channel.

### Invitation redemption creates the account

Invitation codes are generated from cryptographically secure random bytes and stored only as hashes. Each code expires seven days after creation, can be disabled, and can be redeemed exactly once. Redemption and account creation occur in one transaction. The used invitation records the created user and use time.

### WeChat registration is a two-stage authentication

The server exchanges the WeChat code for OpenID. An OpenID linked to a complete account receives an application JWT immediately. Otherwise the server returns a ten-minute registration token containing only the OpenID. That token cannot authorize business APIs.

The user supplies invitation code, username, nickname, and password. Successful transactional redemption creates the shared account with the OpenID. Subsequent mini-program sessions login directly.

### H5 uses the same credentials

H5 registration redeems an invitation without an OpenID. H5 login uses username and password and returns the same account data used after WeChat linking. Passwords are 8-72 bytes. Login errors do not reveal whether the username exists.

### Administrators manage invitations

Authenticated administrators can generate 1-100 invitations in one request, list newest invitations with status, and disable an unused invitation. Plain codes are returned only by the generation response; listings expose a safe prefix and metadata, not reusable secrets.

## Migration Plan

1. Add username/password and invitation records.
2. Replace the old mandatory OpenID uniqueness rule with a partial non-empty index.
3. Deploy admin invitation endpoints and UI.
4. Deploy H5 registration/login and WeChat registration-token behavior.
5. Generate initial invitations and clear test identities before public release.

## Risks / Trade-offs

- Invitation leakage permits one unauthorized registration -> single use, seven-day expiry, disabling, secure randomness, and audit metadata.
- No self-service recovery -> administrators must reset credentials until a verified recovery channel is added.
- Username squatting -> invitations constrain registration and normalized uniqueness prevents lookalike casing duplicates.
- A stolen registration token can be replayed briefly -> expire in ten minutes and reject OpenID conflicts transactionally.
