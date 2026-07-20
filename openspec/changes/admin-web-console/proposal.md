# Admin Web Console

## Why

The mini program should be a focused user-facing study tool. Admin actions such as user banning, reward configuration, notification template setup, and operational review do not belong in the mini program experience and may increase review and usability risk.

## What Changes

- Remove the admin page and admin entry points from the mini program.
- Add a simple PC web admin console built with Vue.
- Reuse existing backend admin APIs and admin role authorization.
- Add username/password admin login and session handling suitable for desktop web usage.
- Support MVP admin workflows: overview, user management, role visibility, ban/unban, slack configuration, AI model configuration, WeChat subscription message configuration, and audit logs.
- Move management-only surfaces out of the mini program and into the PC admin console.
- Add audit logging for sensitive admin actions.

## Non-Goals

- No complex enterprise RBAC in this iteration.
- No mobile admin app.
- No public registration for admin users.
- No analytics-heavy BI dashboard.
- No WeChat QR-code login in this iteration.
- No SMS or phone-code login in this iteration.
- No role editing beyond the existing admin/normal-user model in this iteration.

## Impact

This change separates user and admin surfaces. The mini program becomes cleaner for WeChat review and end users, while admin workflows move to a PC-oriented web console.
