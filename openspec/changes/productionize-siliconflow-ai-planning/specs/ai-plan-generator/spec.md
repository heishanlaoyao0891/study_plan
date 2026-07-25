## MODIFIED Requirements

### Requirement: AI provider is configurable
The system SHALL allow administrators to configure an explicit supported provider and SHALL validate that configuration before it is used for plan generation.

#### Scenario: Configure SiliconFlow provider
- **WHEN** admin selects the `siliconflow` preset
- **THEN** the console recommends Base URL `https://api.siliconflow.cn/v1` and model `deepseek-ai/DeepSeek-V3.2`
- **AND** the backend uses the OpenAI-compatible chat-completions transport without changing the configured provider identity

#### Scenario: Configure generic OpenAI-compatible provider
- **WHEN** admin selects `openai_compatible` and provides model, Base URL, API key, timeout, daily limit, and enabled state
- **THEN** the backend validates and saves the supported configuration for future generation requests

#### Scenario: Load a legacy OpenAI provider alias
- **WHEN** a stored or submitted provider is `openai` or `openai-compatible`
- **THEN** the backend normalizes and persists it as `openai_compatible` rather than falling back

#### Scenario: Load a legacy direct DeepSeek configuration
- **WHEN** a persisted provider is `deepseek` and its Base URL is empty or the canonical direct URL `https://api.deepseek.com`
- **THEN** the backend normalizes and persists it as `siliconflow`
- **AND** sets Base URL to `https://api.siliconflow.cn/v1`
- **AND** replaces model `deepseek-chat` only when that old default is present and otherwise preserves the model

#### Scenario: Load a legacy DeepSeek custom gateway
- **WHEN** a persisted provider is `deepseek` and its Base URL is a custom gateway URL
- **THEN** the backend normalizes and persists it as `openai_compatible`
- **AND** preserves both the custom Base URL and configured model, including `deepseek-chat`

#### Scenario: Reject an unsafe provider destination
- **WHEN** a generic OpenAI-compatible Base URL targets or resolves to localhost, loopback, private, link-local, unspecified, or multicast addresses
- **THEN** the backend rejects it before sending credentials or request content

#### Scenario: Reject unknown provider
- **WHEN** a configuration names a provider other than `siliconflow`, `openai_compatible`, or `mock`
- **THEN** the backend rejects it and does not silently substitute mock output

#### Scenario: Test provider planning compatibility
- **WHEN** admin runs an enabled real-provider connection test
- **THEN** the backend requests a minimal structured plan and reports success only if authentication, completion decoding, plan parsing, and production schema validation all succeed

### Requirement: AI generation has fallback behavior
The system SHALL provide a deterministic fallback when an enabled provider fails while truthfully identifying whether content came from AI or fallback rules.

#### Scenario: Provider request fails
- **WHEN** the configured provider times out, rejects the request, or cannot be decoded
- **THEN** the backend may return a valid deterministic preview with `mode=fallback`
- **AND** the response identifies the configured provider and a bounded fallback reason without exposing secrets

#### Scenario: Provider returns an oversized response
- **WHEN** the provider response body exceeds the configured one-megabyte safety bound
- **THEN** the backend stops reading and treats the response as a provider failure

#### Scenario: Provider returns invalid structured output
- **WHEN** the provider response fails plan parsing or business validation
- **THEN** the backend returns a validated deterministic preview marked as fallback instead of presenting it as AI output

#### Scenario: AI is disabled
- **WHEN** a user requests generation while AI is disabled
- **THEN** the backend rejects the request with a clear disabled status and does not label rule output as AI

## ADDED Requirements

### Requirement: AI credentials are encrypted at rest
The system SHALL protect stored real-provider API keys with server-side authenticated encryption and SHALL never return complete key material through an API.

#### Scenario: Save API key
- **WHEN** admin saves a non-empty AI API key
- **THEN** the backend encrypts it with `AI_KEY_ENCRYPTION_SECRET` before persistence and returns only fixed masked and storage-state metadata

#### Scenario: Encryption secret is unavailable
- **WHEN** admin attempts to save an API key without `AI_KEY_ENCRYPTION_SECRET`
- **THEN** the backend rejects the update instead of persisting plaintext

#### Scenario: Use encrypted API key
- **WHEN** an enabled real provider sends a completion request
- **THEN** the backend decrypts the key only for the outbound authorization header and does not log or return it

#### Scenario: Startup finds legacy plaintext keys
- **WHEN** startup migration finds one or more non-empty legacy plaintext AI keys and `AI_KEY_ENCRYPTION_SECRET` is available
- **THEN** the backend transactionally encrypts every plaintext key before startup continues

#### Scenario: Startup cannot encrypt a legacy plaintext key
- **WHEN** startup migration finds a non-empty legacy plaintext AI key and the encryption secret is unavailable or encryption or persistence fails
- **THEN** the transaction rolls back and startup fails without leaving a partial migration

#### Scenario: Startup finds no configured AI key
- **WHEN** an AI configuration has no API key
- **THEN** startup migration may continue without `AI_KEY_ENCRYPTION_SECRET`
