# Real AI Plan Generation

## Why

AI plan generation currently demonstrates the intended flow with mock data. To make the feature genuinely useful, the system needs a configurable model-provider layer, a business-specific planning agent, strict structured output validation, retry/fallback behavior, historical learning context, and a user review flow before tasks are committed.

## What Changes

- Add configurable AI provider support that can work with multiple model sources, including low-cost domestic models and third-party relay services.
- Add provider connectivity testing in the PC admin console.
- Add a business-specific AI planning agent instead of directly forwarding raw user prompts to a model.
- Request structured JSON plans from the model.
- Validate AI output against a schema before persisting.
- Return a preview that users can edit before committing.
- Add regeneration with refined instructions.
- Add usage controls with a default per-user daily generation limit of 5, configurable from the PC admin console.
- Include historical completion and study-efficiency data in plan generation.

## Non-Goals

- No paid AI quota system for end users.
- No replacement of manual plan creation.
- No open-ended chatbot that lets users directly chat with the model through our backend.
