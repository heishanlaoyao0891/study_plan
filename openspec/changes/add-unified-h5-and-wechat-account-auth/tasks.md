# Tasks

## 1. Identity Contract

- [x] Define one shared account with optional OpenID and username/password credentials.
- [x] Add partial uniqueness for non-empty OpenID and normalized username.
- [x] Add short-lived WeChat registration tokens that cannot access business APIs.

## 2. Invitations

- [x] Add secure hashed, single-use invitation records with seven-day expiry.
- [x] Add administrator single/bulk generation, listing, and disabling APIs.
- [x] Redeem invitations transactionally with account creation.

## 3. Backend Authentication

- [x] Add H5 invitation registration and password login endpoints.
- [x] Return direct login for linked returning WeChat accounts.
- [x] Require invitation registration for new WeChat identities.
- [x] Keep business APIs inaccessible without a complete account JWT.

## 4. Client Authentication

- [x] Add responsive H5 login and invitation registration flows.
- [x] Keep WeChat login direct for linked accounts and show invitation registration only when required.
- [x] Let an H5-created account link to WeChat with username and password without another invitation.
- [x] Use same-origin API defaults for H5 and production API build configuration for WeChat.

## 5. Admin and Verification

- [x] Add invitation generation, bulk count, status list, copy, and disable controls to the admin console.
- [x] Add backend tests for invitation lifecycle, registration, login, and WeChat direct login.
- [x] Run backend tests, frontend/admin type checks, H5 build, and WeChat build.
- [ ] Generate initial production invitations and complete H5/mini-program acceptance testing.
