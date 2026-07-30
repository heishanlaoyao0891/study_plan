## Context

The plan model stores schedule lists with GORM JSON serialization. Historical rows without list values deserialize to nil slices and are emitted as JSON `null`, while the client contract declares arrays and the plan UI uses array methods for schedule summaries.

## Goals / Non-Goals

**Goals:**

- Return stable array-shaped schedule fields for all plan reads.
- Keep plan list and plan detail usable when a legacy server or cached response supplies null schedule values.
- Avoid changing existing plan dates, schedule meaning, or task generation.

**Non-Goals:**

- Backfilling or rewriting historical database rows.
- Changing the user-selected schedule or inventing missing learning days.

## Decisions

- **Normalize at the API boundary.** Before serializing plan views, convert nil `study_weekdays`, `study_dates`, and schedule overrides to empty slices. This fixes every current client and establishes the API contract without a data migration.
- **Keep client guards.** UI helpers treat only arrays as usable arrays. This protects the rendered list and detail views during rolling deployment, cached data, or future nonconforming callers.

## Risks / Trade-offs

- [An empty schedule can look intentional] → The UI continues to render its existing “未设置学习日” / taskless copy; no schedule is fabricated.
- [New list fields could regress] → The normalization helper is kept next to plan list response construction and covered by a null-serialization regression test.

## Migration Plan

1. Deploy the backend and mini-program build together.
2. Verify an existing plan with null schedule fields renders in the plan tab and detail page.
3. Roll back application code if necessary; no database migration is required.

## Open Questions

- None.
