# Task Scheduling Improvements

## Why

The current task flow supports basic timing, completion, makeup, and postponement, but plan/task boundaries and scheduling behavior are not clear enough. Users need a clearer execution model: plans define long-term goals, tasks define concrete scheduled study sessions, and the app should make postponement, makeup, upcoming schedules, and batch adjustments explicit.

## What Changes

- Add task detail view and calendar/list schedule view.
- Improve postpone/makeup interactions with date and time selection.
- Prevent automatic cross-day sessions by closing active study at midnight and allowing user correction the next day.
- Add batch schedule operations for future tasks.
- Add better conflict/overload warnings using task-level time slots.

## Non-Goals

- No drag-and-drop dependency on desktop-only UI.
- No full calendar sync with external calendars.
- No automatic study session continuing into the next calendar day.
