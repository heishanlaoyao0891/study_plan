# Privacy And Account Controls

## Why

The product collects phone number, learning records, check-ins, and group-visible metrics. Users need clear account controls for phone changes, logout/deactivation, optional data retention, data deletion, and privacy-policy disclosure.

## What Changes

- Add account settings page with masked phone display.
- Allow user-initiated phone number rebinding.
- Add account deactivation/logout flow with choice to retain or delete historical data.
- If user retains data, re-authentication with the same verified phone/openid can restore data.
- If user deletes data, remove or anonymize personal and learning data according to policy.
- Add privacy policy guidance covering phone number, avatar URL/key, study records, AI usage, group metrics, notifications, and admin access.

## Non-Goals

- No PC user login in this change.
- No legal document generation guarantee; final policy should still be reviewed before release.
