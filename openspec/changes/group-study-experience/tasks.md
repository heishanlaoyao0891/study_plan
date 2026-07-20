## 1. Study Group Model And Boundary

- [x] 1.1 Add study group model and group member role fields
- [x] 1.2 Enforce one active study group per user
- [x] 1.3 Add leave group and end group APIs
- [x] 1.4 Add leader-only remove member API
- [x] 1.5 Enforce MVP group size limit of 10 active members
- [x] 1.6 Add leader transfer or require leader to end group before leaving
- [x] 1.7 Support optional group end date and manual leader ending
- [x] 1.8 Allow leader to customize group name
- [x] 1.9 Limit leadership transfer to current active members

## 2. Invitations

- [x] 2.1 Add invitation code model and APIs
- [x] 2.2 Add join-by-code API
- [x] 2.3 Add share payload/link support
- [x] 2.4 Add QR/miniprogram-code generation and caching support
- [x] 2.5 Add revoke or expire behavior
- [x] 2.6 Default invitation expiry to 7 days and allow leader regeneration

## 3. Member Status And Privacy

- [x] 3.1 Add group member list API
- [x] 3.2 Add per-member group daily check-in status API
- [ ] 3.3 Add public/private visibility fields for plans/tasks if needed
- [x] 3.4 Ensure members cannot view other members' private plans/tasks
- [x] 3.5 Update frontend group dashboard view
- [x] 3.6 Default group-visible metrics to streak, study minutes, completion rate, level, and current-day completion state
- [ ] 3.7 Add plan/task detail public-to-group switch where applicable

## 4. Level System And Leaderboard

- [x] 4.1 Define member level rules based on check-ins and streaks
- [x] 4.2 Add group leaderboard API
- [x] 4.3 Rank by continuous days, study minutes, completion rate, and level
- [x] 4.4 Add frontend member level and leaderboard section
- [x] 4.5 Implement weekly and all-time leaderboard scopes
- [ ] 4.6 Add historical ended-group detail view

## 5. Nudges

- [ ] 5.1 Add member nudge endpoint
- [ ] 5.2 Connect nudge to WeChat subscription notification event queue
- [ ] 5.3 Record nudge attempts when target user has not subscribed
- [ ] 5.4 Add frontend nudge action
- [ ] 5.5 Limit nudges to same target once per day and max 3 received nudges per day

## 6. Verification

- [ ] 6.1 Verify backend build
- [ ] 6.2 Verify mini program build
- [ ] 6.3 Add focused access-control tests for group privacy and one-active-group rule
