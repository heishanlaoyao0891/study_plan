## Why

AI planning accepts generic OpenAI-compatible text configuration, but production behavior is unsafe and difficult to operate. Encrypted keys are not decrypted before use, missing encryption configuration permits plaintext storage, unknown providers silently become mock output, connection tests only request ping text, and fallback responses do not explain why AI was not used. SiliconFlow also lacks an operator-safe preset despite being the recommended provider.

## What Changes

- Add explicit `siliconflow`, `openai_compatible`, and `mock` provider modes with strict validation and no silent provider substitution.
- Add a SiliconFlow admin preset using canonical Base URL `https://api.siliconflow.cn/v1` and recommended model `deepseek-ai/DeepSeek-V3.2`.
- Migrate legacy persisted direct `deepseek` rows to `siliconflow`, while moving custom-gateway rows to `openai_compatible` without changing their URL or model.
- Encrypt API keys at rest with `AI_KEY_ENCRYPTION_SECRET`, decrypt only for outbound requests, never return key material, and transactionally migrate every legacy plaintext key during startup or fail startup.
- Replace ping testing with a minimal structured-plan generation that must pass the production plan parser and validator.
- Expose configured provider, model, effective mode, key-storage health, and explicit fallback reasons to operators and generation clients.
- Preserve deterministic fallback planning while ensuring fallback is never represented as AI output.

## Capabilities

### Modified Capabilities

- `ai-plan-generator`: Define provider presets, structured connection validation, encrypted credentials, and truthful AI/fallback result metadata.
- `admin-config`: Add guided SiliconFlow setup, strict configuration validation, masked key state, and observable effective mode.

## Non-Goals

- No real API key is included or requested by the repository.
- No new end-user AI configuration or general chatbot.
- No SiliconFlow-specific request options beyond its documented OpenAI-compatible chat-completions interface.
