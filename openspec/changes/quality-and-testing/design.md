# Design: Quality And Testing

## Backend Tests

Use Go tests around handlers and database behavior with isolated test databases. Cover authorization, ownership checks, task lifecycle, check-in rewards, stats, admin-only endpoints, and important scheduling edge cases. The goal is focused regression coverage, not 100% line coverage.

## Database Reliability

Add explicit indexes/constraints for user-plan-date and ownership-sensitive relations. Keep migrations automatic for now, but verify startup migration behavior.

## Frontend Verification

Keep the required baseline as mini program build/type-check plus targeted critical flow checklist. The PC admin console should also run build/type-check once it exists. Add H5 smoke testing only when it catches real integration issues.

## Release Gates

Define a local PowerShell verification script that runs backend tests, backend build, mini program type-check/build, admin console type-check/build if present, and OpenSpec validation for active changes. GitHub Actions can be added later, but local verification is the required baseline.

## Critical Flow Checklist

Keep a lightweight manual checklist for mini program preview:

- Login and phone binding.
- Create plan.
- Generate AI preview and commit.
- Start/stop/complete task.
- Auto check-in and slack reward.
- Postpone and makeup task.
- Group join and nudge.
