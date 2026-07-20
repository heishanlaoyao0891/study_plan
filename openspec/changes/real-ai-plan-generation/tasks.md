## 1. Provider Integration

- [x] 1.1 Add AI provider configuration and provider interface
- [x] 1.2 Implement mock provider behind the interface
- [x] 1.3 Implement OpenAI-compatible provider for domestic models and relay services
- [x] 1.4 Add provider connectivity test API for PC admin console
- [x] 1.5 Connect provider config to admin-managed AI settings

## 2. Planning Agent

- [x] 2.1 Build controlled planning prompt from product-specific inputs
- [x] 2.2 Include user historical completion rate and study efficiency in agent context
- [x] 2.3 Add deterministic rule-based fallback planner
- [x] 2.4 Return planning rationale with generated preview
- [x] 2.5 Define allowlisted planning tools for learning profile, active load, task outcomes, and schedule conflict checks
- [x] 2.6 Implement planning tools as backend-owned Go functions scoped to the authenticated user
- [x] 2.7 If an Agent framework is introduced, wrap only the allowlisted tools and keep final validation in backend code

## 3. Structured Plan Output

- [x] 3.1 Define AI plan JSON schema
- [x] 3.2 Validate model output before returning it
- [x] 3.3 Repair minor JSON format issues when safe
- [x] 3.4 Reject invalid dates, empty tasks, and excessive durations
- [x] 3.5 Enforce default maximum preview length of 30 days

## 4. Preview And Commit

- [x] 4.1 Change generation to return editable preview
- [x] 4.2 Add commit endpoint to persist accepted preview
- [x] 4.3 Update frontend AI page with preview editing before save
- [x] 4.4 Support regeneration with user refinements
- [x] 4.5 Allow editing title, description, date, time, estimated minutes, and difficulty in preview

## 5. Reliability And Cost

- [x] 5.1 Add timeout and retry behavior
- [x] 5.2 Add per-user generation limits with default 5/day
- [x] 5.3 Make generation limit configurable from PC admin console
- [x] 5.4 Add clear fallback errors when provider is unavailable
- [x] 5.5 Track generation usage and provider failures
- [x] 5.6 Ensure fallback planner produces a legal preview without provider access

## 6. Verification

- [ ] 6.1 Verify backend build
- [ ] 6.2 Verify mini program build
- [ ] 6.3 Verify admin console AI config/test flow if admin console is present
- [ ] 6.4 Add focused API tests for schema validation and fallback planner
