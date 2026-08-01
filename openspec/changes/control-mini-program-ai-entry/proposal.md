## Why

The current mini-program always exposes AI plan generation whenever the shared AI provider is enabled, but the current WeChat account cannot publish that capability. The operator needs a fail-closed runtime switch in the PC admin console so a reviewed mini-program can expose only manual planning while H5 continues using AI planning.

## What Changes

- Add an independent `mini_program_ai_enabled` setting to the existing AI administration configuration, defaulting to disabled.
- Expose a minimal authenticated client-feature response that contains no provider credentials or internal AI settings.
- Hide the mini-program AI hero, AI job state, polling, and navigation while the setting is disabled; promote manual plan creation as the primary action.
- Reject mini-program AI plan-job reads and submissions while disabled, while leaving H5 behavior unchanged.
- Add an explicit compliance-risk warning and audit trail when administrators change the mini-program switch.

## Capabilities

### New Capabilities

- `client-feature-control`: Runtime, channel-specific feature availability delivered fail-closed to authenticated clients.

### Modified Capabilities

- `admin-config`: Administrators can independently control mini-program AI plan availability without disabling the provider or H5 AI planning.
- `ai-plan-generator`: AI plan entry, job access, and generation are conditioned on the mini-program channel setting.

## Impact

- Backend AI configuration model, migrations, admin API, audit records, client-feature API, and AI plan-job handlers.
- PC admin AI configuration form and API types.
- UniApp request channel metadata, plan page, and AI generation page.
- Backend, admin, and mini-program regression tests plus release documentation.
