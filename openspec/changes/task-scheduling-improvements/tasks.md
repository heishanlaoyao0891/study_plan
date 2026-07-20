## 1. Schedule Fields

- [x] 1.1 Add planned start/end fields to daily tasks
- [x] 1.2 Update task generation to set planned times
- [x] 1.3 Update overload checks to consider planned times
- [x] 1.4 Update UI/API wording to distinguish plan as goal container and task as scheduled execution unit
- [x] 1.5 Add `needs_decision` state or equivalent field for midnight auto-closed tasks

## 2. Calendar And Detail UI

- [x] 2.1 Add task detail page
- [x] 2.2 Add future 7-day schedule list view
- [x] 2.3 Add quick actions for start, complete, postpone, and makeup
- [x] 2.4 Future 7-day list includes today plus the next 6 days

## 3. Postpone And Makeup

- [x] 3.1 Improve postpone API to include target date and planned start/end time
- [x] 3.2 Improve frontend date/time picker interaction
- [x] 3.3 Add postpone history display
- [x] 3.4 Improve makeup API to allow editing actual start and actual end time
- [x] 3.5 Recalculate study minutes after makeup start/end edits
- [x] 3.6 Enforce makeup constraints: end after start, no future end, max 8 hours per corrected session
- [x] 3.7 Warn but allow confirmed postpone into conflicting planned time

## 4. Midnight Boundary Handling

- [x] 4.1 Send or surface final 23:30 active-task reminder
- [x] 4.2 Auto-close active sessions at 00:00 and attribute them to the previous day's task
- [x] 4.3 Allow next-day manual correction of actual end time past midnight while keeping minutes attributed to the previous day
- [x] 4.4 Verify stats include midnight-corrected minutes correctly
- [x] 4.5 Use `Asia/Shanghai` as scheduling timezone
- [x] 4.6 Add frontend compensation check for missed midnight decisions when user opens mini program

## 5. Batch Operations

- [x] 5.1 Support shifting future tasks by a number of days
- [x] 5.2 Preserve planned time ranges during batch shift
- [x] 5.3 Add frontend batch shift action for future tasks
- [x] 5.4 Default batch shift start date to tomorrow and only move unfinished tasks

## 6. Completion And Check-In

- [x] 6.1 Auto-complete plan/date check-in when all tasks for that plan/date are completed
- [x] 6.2 Keep check-in reward behavior idempotent when auto-completed from tasks

## 7. Verification

- [x] 7.1 Verify backend build
- [x] 7.2 Verify mini program build
- [x] 7.3 Add focused tests for postpone, makeup, midnight boundary, batch shift, and auto-check-in rules
