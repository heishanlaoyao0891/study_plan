## Why

The product currently authenticates only through WeChat, which prevents users from accessing the same learning data from a normal browser. Phone and email verification add provider qualification, delivery, and recurring operational work that is not justified for the first small-scale release. A controlled invitation system can provide one shared H5 and WeChat account without external identity providers.

## What Changes

- Build and deploy the existing UniApp user experience as H5 with username/password authentication.
- Let administrators generate one or many single-use invitation codes that expire after seven days and can be disabled before use.
- Require invitation code, unique username, nickname, and password when creating an account from H5 or the mini program.
- Let a user who registered on H5 link that existing account to WeChat with its username and password without consuming another invitation.
- Treat the application account as the shared user identity and allow one optional WeChat OpenID on it.
- Return a short-lived registration token instead of an application JWT when a WeChat OpenID has no account.
- Let returning WeChat users whose OpenID is linked to a complete account login directly without repeated input.
- Remove phone, email, SMS, and verification-provider dependencies from this release.

## Capabilities

### Modified Capabilities

- `user-auth`: Add invitation-controlled H5 registration/login and WeChat account creation.
- `admin-config`: Add invitation-code generation, listing, and disabling.

## Impact

- User schema, invitation schema, authentication middleware, JWT contracts, and database indexes.
- H5 and mini-program login experience, admin console, and production builds.
- Password recovery is an administrator-assisted operation until a verified recovery channel is introduced.

## Confirmed Decisions

- H5 reuses all current UniApp user features.
- Usernames contain 4-24 letters, digits, or underscores and are unique case-insensitively.
- H5 registration uses invitation code, username, nickname, and password; normal login uses username and password.
- Mini-program first use requires the same four fields and binds the resulting account to the current OpenID.
- A returning linked OpenID logs in immediately.
- Invitations are single-use, expire after seven days, support single or bulk generation, expose use status, and can be disabled.
