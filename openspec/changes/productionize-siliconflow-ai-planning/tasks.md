# Tasks

## 1. Provider And Security
- [x] Add explicit provider identifiers and SiliconFlow defaults.
- [x] Centralize AES-GCM API-key encryption and decryption.
- [x] Reject plaintext key persistence and migrate legacy plaintext when possible.
- [x] Validate provider, URL, model, timeout, limit, enabled state, and credentials.
- [x] Pin SiliconFlow and block private or local OpenAI-compatible destinations at validation and dial time.
- [x] Require a new test key when the tested provider origin changes.
- [x] Normalize legacy OpenAI aliases and carefully migrate legacy direct DeepSeek configuration.
- [x] Bound provider response-body reads.

## 2. Validation And Observability
- [x] Test providers with a validated structured plan response.
- [x] Expose effective provider mode and masked key-storage health.
- [x] Label deterministic fallback responses and usage records truthfully.

## 3. Admin Experience
- [x] Add guided SiliconFlow provider selection and recommended defaults.
- [x] Explain structured testing, encryption state, and fallback behavior.
- [x] Preserve write-only API-key updates and display actionable failures.

## 4. Verification
- [x] Add focused provider, encryption, validation, and structured-test coverage.
- [x] Add SSRF, stored-key reuse, alias migration, and response-limit regressions.
- [x] Run relevant and full backend tests and build.
- [x] Run admin type-check and production build.
- [x] Strictly validate this OpenSpec change.
