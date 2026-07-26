## MODIFIED Requirements

### Requirement: AI generates a study plan from user description
The system SHALL produce a deterministic valid local plan before optional model enrichment, SHALL bound enrichment within the interactive request budget, and SHALL keep user-facing plan introductions in Simplified Chinese.

#### Scenario: Model enrichment exceeds its budget
- **WHEN** the configured model does not finish enrichment within the bounded enrichment window
- **THEN** the system returns the complete local plan with timeout metadata instead of failing generation

#### Scenario: Local plan is generated
- **WHEN** model enrichment is disabled, unavailable, invalid, or times out
- **THEN** the returned summary, rationale, and capacity warnings are Chinese user-facing copy

#### Scenario: Model returns an English introduction
- **WHEN** enrichment returns English summary or rationale fields
- **THEN** the server preserves the canonical Chinese local summary and rationale
- **AND** H5 and mini-program clients do not display that English introduction
