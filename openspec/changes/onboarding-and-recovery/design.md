# Design: Onboarding And Recovery

## First-Run Flow

The first-run flow should be short and action-oriented:

1. Bind phone number.
2. Optionally enable reminders with clear value copy.
3. Create a plan manually or generate one with AI.
4. Show today's task and start button.

Users should be able to skip non-required steps. Phone binding remains required by the production-hardening change.

## Reminder Subscription UX

Subscription prompts should be contextual and respectful:

- Ask at setup for recommended reminders.
- Ask again only when the user enables reminder settings or performs a reminder-relevant action.
- Do not repeatedly prompt after refusal.
- Provide a reminders settings page to retry authorization.

## Recovery Flow

When the system detects missed days, overdue tasks, or many pending decisions, it should offer a recovery entry such as `我落后了，帮我重新安排`. The flow previews a revised future schedule and requires user confirmation before applying changes.

Recovery can use AI when available and rule-based scheduling when AI is disabled or unavailable.
