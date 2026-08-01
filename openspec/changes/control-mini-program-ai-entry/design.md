## Context

Version 1.0.1 is already published with AI planning and does not understand remote client-feature configuration. A subsequent mini-program release must be able to hide or restore AI planning at runtime without disabling H5 AI or changing provider credentials. The setting is operational control, not proof of platform eligibility, and changing it after review carries platform-policy risk.

## Goals / Non-Goals

**Goals:**

- Add a fail-closed mini-program AI planning switch managed from the existing PC AI configuration page.
- Keep H5 AI generation and the underlying provider configuration independent.
- Make supported mini-program versions hide all AI planning entry/status UI and stop polling while disabled.
- Reject AI plan-job access from supported mini-program versions while disabled.
- Preserve the behavior of already-published clients that do not send channel metadata or read feature flags.

**Non-Goals:**

- This change does not guarantee WeChat review approval or authorize use of a restricted capability.
- It does not remotely remove bundled code from an already-built package.
- It does not alter AI generation quality, quotas, provider configuration, or H5 planning behavior.

## Decisions

### Store the channel switch beside provider configuration

Add `mini_program_ai_enabled` to `AIConfig`, defaulting to false. It is returned and updated through the existing admin AI configuration endpoints. This keeps operational ownership in one screen while preserving `enabled` as the provider-wide service switch.

Alternative considered: reuse `AIConfig.Enabled`. Rejected because it would also disable H5 and background AI processing.

### Expose only effective client features

Add an authenticated client-feature endpoint returning `mini_program_ai_enabled` without provider names, URLs, keys, quotas, or other administration data. Missing configuration or query failure resolves to false.

### Mark supported requests with compile-time channel metadata

The shared UniApp request layer sends `X-Client-Platform: mp-weixin` in the WeChat build and `h5` in H5. AI plan-job endpoints enforce the mini-program switch only for explicit `mp-weixin` requests. This makes the new client behavior deterministic while preserving version 1.0.1, which sends no channel metadata.

Alternative considered: infer channel from User-Agent or Referer. Rejected because those values are proxy-dependent and can misclassify WeChat H5 traffic.

### Hide and guard the complete client workflow

When disabled, the plan page omits the AI hero, job status, current-job request, and polling. Manual creation becomes the visible primary command. Direct entry to the AI page checks the same feature endpoint and returns to the plan page before allowing submission. Backend enforcement prevents a supported client from bypassing visibility through direct navigation.

### Audit administrative changes

Changing the switch uses the existing `update_ai_config` audit action with a reason that includes the old and new mini-program state. The admin UI shows that enabling after approval can trigger platform review or enforcement.

## Risks / Trade-offs

- [Remote activation can violate platform policy] -> Display an explicit admin warning and keep the default disabled; the operator owns the activation decision.
- [Published version 1.0.1 does not read the switch] -> Preserve its existing behavior intentionally; the switch governs clients that implement the new feature contract.
- [Feature fetch fails during startup] -> Default to hidden and do not call AI job APIs.
- [Configuration changes while a page is open] -> Refresh features on each page show; active jobs continue server-side but UI follows the latest switch.
- [Channel headers are forgeable] -> Treat this as product gating rather than authorization or a security boundary.

## Migration Plan

1. Add the nullable/default-false database column through GORM migration and return it from admin APIs.
2. Deploy backend and admin with the switch left off.
3. Submit the new mini-program version; it hides AI by default.
4. After release, the operator may change the runtime switch from the admin console.
5. Roll back by switching it off; H5 and provider configuration remain unchanged.

## Open Questions

None. The operator explicitly accepts runtime activation and its platform-policy risk.
