# Quality And Testing

## Why

The app now has enough behavior that regressions are likely without focused tests and stricter data guarantees. Before further feature growth, add a pragmatic quality layer around backend APIs, database constraints, frontend critical flows, and release checks.

## What Changes

- Add focused backend API tests for auth, plans, tasks, check-ins, slack, stats, and admin access.
- Add database constraints and migration checks where missing.
- Add mini program and PC admin console build/type checks.
- Add local PowerShell verification script for repeatable checks.
- Document release gates.

## Non-Goals

- No exhaustive end-to-end automation for every mini program screen.
- No large framework migration.
- No mandatory GitHub Actions/CI setup in this iteration; local verification is the baseline.
- No full frontend UI automation suite in this iteration.
