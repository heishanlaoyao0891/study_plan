## MODIFIED Requirements

### Requirement: New user receives guided first-run onboarding
The system SHALL guide new users through unique nickname setup, optional reminders, AI-assisted or manual first-plan creation, and today's first task without requesting phone binding.

#### Scenario: New user starts app
- **WHEN** a new user completes WeChat authentication
- **THEN** the first required step is choosing a valid unique nickname
- **AND** phone binding is not shown as an onboarding step

#### Scenario: Nickname setup completes
- **WHEN** the user saves an available nickname
- **THEN** onboarding promotes `AI 生成第一个计划` as the primary next action
- **AND** offers manual creation as a secondary choice

#### Scenario: Existing incomplete profile returns
- **WHEN** a user with no valid unique nickname reopens the mini program
- **THEN** onboarding resumes at nickname setup without losing existing learning data
