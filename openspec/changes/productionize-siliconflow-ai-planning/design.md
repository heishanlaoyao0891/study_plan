## Context

The provider layer already calls `/v1/chat/completions` and validates plan previews, while the admin console persists one AI configuration row. The existing AES-GCM write path lives in an HTTP handler and marks ciphertext as encrypted, but the provider returns ciphertext unchanged instead of decrypting it. If the server secret is absent, it stores plaintext. Unknown provider names resolve to mock, and provider failure triggers a valid deterministic preview without enough source metadata.

## Decisions

### Provider identifiers are explicit

The backend accepts only `siliconflow`, `openai_compatible`, and `mock`. Legacy `openai` and `openai-compatible` identifiers are migrated and normalized to `openai_compatible` on load and update. Legacy `deepseek` identifiers with an empty Base URL or the old canonical direct URL `https://api.deepseek.com` normalize to `siliconflow` and its canonical URL; model `deepseek-chat` is replaced only for those direct-provider rows. Legacy `deepseek` rows with a custom gateway normalize to `openai_compatible`, preserving both URL and model because `deepseek-chat` may remain valid at that gateway. SiliconFlow reuses the OpenAI-compatible transport but remains a distinct operational identifier and pins Base URL to exactly `https://api.siliconflow.cn/v1`; its admin preset recommends model `deepseek-ai/DeepSeek-V3.2`. Unknown values fail validation and use an unsupported provider error rather than silently selecting mock.

### Configuration validation is shared

Save and test paths share validation for provider, model, absolute HTTP(S) Base URL, timeout, daily limit, and API-key presence. Production requires HTTPS and `AI_KEY_ENCRYPTION_SECRET`. Generic OpenAI-compatible URLs reject credentials, query strings, fragments, localhost, and literal or DNS-resolved loopback, private, link-local, unspecified, or multicast destinations. SiliconFlow requires the canonical HTTPS origin and `/v1` path. The transport repeats public-address validation when dialing and validates redirects to prevent DNS rebinding. Both host roots and URLs ending in `/v1` resolve to exactly one `/v1/chat/completions` endpoint.

### Keys use shared AES-GCM handling

Generic API-key encryption and decryption live in a small dependency package shared by database migration and service wrappers. New real-provider keys cannot be persisted without `AI_KEY_ENCRYPTION_SECRET`; API responses return only a fixed mask and storage health. During startup, all non-empty legacy plaintext rows are encrypted in one transaction. A missing secret or any encryption or update failure rolls back the transaction and fails startup; configurations without a key do not require a secret for migration. Existing encrypted rows require the same secret to decrypt, so operators must preserve and back up that environment secret.

### Connection tests prove planning compatibility

A provider test requests a one-task JSON plan, parses it with the production parser, and validates it with the production schema. Authentication, endpoint compatibility, completion decoding, JSON shape, and required task fields therefore all participate in the test. The test never persists generated content or credentials.

An admin test may reuse the stored key only when the tested Base URL has the same scheme and authority as the persisted Base URL. Testing another origin requires a newly supplied write-only key, preventing a changed test URL from receiving persisted credentials. Provider response bodies are capped at 1 MiB before decoding.

### Fallback remains safe and truthful

Provider transport errors and invalid provider output produce the deterministic preview. The response reports `mode=fallback`, configured provider/model, and a bounded fallback reason; usage records use a fallback status. Disabled AI remains a clear rejection. Mock mode is an explicit rule-fallback choice rather than an implied AI provider.

## Risks / Trade-offs

- Requiring the encryption secret in production may block startup for deployments that omitted it; this is intentional to prevent future plaintext credentials.
- Existing encrypted keys cannot be recovered if the original encryption secret is lost; operators must enter a replacement key after restoring the correct secret.
- Structured tests consume a small provider request instead of a cheaper ping, but they detect materially more production incompatibilities.

## Migration Plan

1. Configure a strong stable `AI_KEY_ENCRYPTION_SECRET` before deployment.
2. Deploy the backend so startup transactionally re-encrypts all legacy plaintext keys before serving requests.
3. Select SiliconFlow in admin, enter the key through the write-only field, save, and run the structured test.
4. Confirm effective AI mode and fallback records before enabling normal traffic.
