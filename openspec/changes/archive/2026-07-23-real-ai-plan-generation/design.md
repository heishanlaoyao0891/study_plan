# Design: Real AI Generation

## Provider Layer

The backend should expose an internal provider interface so the app can run with `mock`, OpenAI-compatible, or provider-specific adapters without changing handler code.

OpenAI-compatible means the provider exposes an API shaped like OpenAI chat/completions APIs: configurable base URL, API key, model name, messages array, temperature, timeout, and JSON response handling. Many domestic models and relay providers support this style, even when the actual model is not OpenAI.

Provider configuration should live in the PC admin console and include:

- Provider type: `mock`, `openai_compatible`, or future provider-specific adapters.
- Base URL.
- Model name.
- API key, stored securely and returned only as masked status.
- Request timeout.
- Daily per-user generation limit, default `5`.
- Enabled/disabled state.

The admin console should support a provider connectivity test that sends a minimal validation request and reports whether credentials, base URL, model name, and response parsing work.

## Planning Agent

The backend should act as a business-specific planning agent. It should not pass raw user prompts directly to the model. The agent builds a controlled prompt from:

- User goal.
- Available study time and preferred time slots.
- Start date and skip dates.
- User historical completion rate.
- Average study minutes.
- Recent unfinished/postponed task patterns.
- Product constraints such as maximum daily load and required JSON schema.

The agent should produce a plan preview, not persist tasks immediately. It should also explain the planning rationale briefly so users understand why the schedule looks the way it does.

## Controlled Tool Calling

The planning agent MAY use tool calling, but only through backend-owned allowlisted tools. These tools should expose business facts, not raw database access. Example tools:

- `get_user_learning_profile`: returns completion rate, average study minutes, recent streak, and postpone frequency.
- `get_active_plan_load`: returns active plan count and weekly target hours.
- `get_recent_task_outcomes`: returns aggregated completed, missed, postponed, and makeup task counts.
- `check_schedule_conflicts`: checks whether proposed tasks conflict with existing planned tasks.

Tool outputs must be scoped to the authenticated user. The model must not receive credentials, SQL, raw tokens, or unrestricted query capability. The backend remains responsible for validating all tool outputs and final plan previews.

For MVP, the service MAY implement these as normal Go functions called by the planning service before or during generation. If an Agent framework is introduced, it should wrap these same functions instead of changing business ownership.

## Structured Output

The AI request asks for JSON containing plan metadata and task items. The backend validates required fields, dates, duration limits, and skip dates before returning a preview.

The schema should include at least plan title, summary, estimated total hours, daily tasks, planned date/time, task title, task description, estimated minutes, and difficulty. The backend may repair minor format issues, but invalid or unsafe output must not be persisted.

The default maximum generated preview length should be 30 days. Longer plans should require explicit configuration or regeneration in phases.

## Preview Before Commit

Generation should not immediately create permanent tasks by default. The user reviews and edits the preview, then commits it to a plan.

The preview should allow editing task title, description, date, planned start/end time, estimated minutes, and difficulty before commit.

## Cost Controls

The backend should enforce per-user generation limits, input length limits, and timeout settings.

The default generation limit is 5 per user per day and should be configurable from the PC admin console.

## Failure And Fallback

Failures can occur because providers timeout, relay services reject credentials, rate limits are hit, JSON is malformed, model output violates business constraints, or network access fails. The backend should retry once for transient provider/output issues, attempt schema repair when safe, and then fall back to a deterministic rule-based planner when necessary. The user should receive either a valid editable preview or a clear error explaining why generation is unavailable.

The deterministic fallback planner must at minimum generate a legal preview from user goal, start date, available minutes, requested days, and skip dates, respecting the 30-day default maximum and workload limits.
