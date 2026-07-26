## MODIFIED Requirements

### Requirement: User can record slacking activity
The system SHALL settle an ended slack session using its full elapsed minutes, SHALL allow that settlement to create a signed negative balance, and SHALL block new sessions while the balance is non-positive.

#### Scenario: Session exceeds available balance
- **WHEN** a valid active session consumes more minutes than the current balance
- **THEN** settlement records the exact negative delta and the resulting balance becomes negative

#### Scenario: User has slack debt
- **WHEN** the user's balance is zero or negative
- **THEN** the backend and client prevent another slack session and explain that task completion plus daily check-in can repay the debt

#### Scenario: Check-in reward repays debt
- **WHEN** a user with a negative balance earns the normal daily check-in reward
- **THEN** the reward increases the signed balance and reduces or clears the debt
