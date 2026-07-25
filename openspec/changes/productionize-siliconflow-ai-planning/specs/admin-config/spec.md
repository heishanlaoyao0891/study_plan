## MODIFIED Requirements

### Requirement: Admin can configure AI model access
The system SHALL provide guided, validated, and observable AI provider configuration in the PC admin console.

#### Scenario: Select SiliconFlow preset
- **WHEN** admin selects SiliconFlow in the AI configuration page
- **THEN** the page fills canonical Base URL `https://api.siliconflow.cn/v1` and recommended model `deepseek-ai/DeepSeek-V3.2`
- **AND** the backend rejects any other SiliconFlow Base URL

#### Scenario: Save production-ready configuration
- **WHEN** admin saves an enabled real-provider configuration
- **THEN** the backend validates provider, model, Base URL, timeout, daily limit, and API-key availability before persisting it

#### Scenario: Update AI API key
- **WHEN** admin supplies a replacement API key
- **THEN** the system encrypts the key at rest and later returns only a fixed mask plus `missing`, `plaintext`, or `encrypted` storage health

#### Scenario: Preserve AI API key
- **WHEN** admin saves configuration with the API-key field empty
- **THEN** the existing encrypted key is preserved and never sent to the browser

#### Scenario: View effective provider mode
- **WHEN** admin opens AI configuration
- **THEN** the console shows whether effective operation is AI, rule fallback, or disabled and explains deterministic fallback behavior

#### Scenario: Run structured provider test
- **WHEN** admin tests an enabled SiliconFlow or OpenAI-compatible configuration
- **THEN** the console reports whether a production-valid structured plan was returned rather than reporting a text ping as success

#### Scenario: Test a different provider origin
- **WHEN** admin changes the scheme or authority from the persisted Base URL while testing
- **THEN** the backend requires a newly supplied API key and never sends the persisted key to that origin
