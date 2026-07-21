# Design: Privacy And Account Controls

## Account Settings

The mini program should show a small, non-prominent masked phone number in account/settings, for example `138****5678`, and provide a `Change phone number` action.

## Phone Rebinding

Phone rebinding uses the WeChat phone authorization flow again. After successful verification, the old phone is replaced and an audit-style account event is recorded.

## Account Deactivation

User logout/deactivation should ask whether to retain data:

- Retain data: mark account inactive or logged out while preserving plans, tasks, check-ins, study sessions, slack records, and group history for future restoration.
- Delete data: delete or anonymize user-owned personal and learning data according to documented policy.

When retained data exists, a future login with the same verified identity should restore the account data.

## Privacy Policy Guidance

Privacy policy should explain:

- Why phone number is required.
- What avatar data is stored and where images live.
- How study plans, tasks, check-ins, study sessions, and slack records are used.
- How AI generation may use user goals and aggregated historical learning data.
- What group-visible data is shared with group members.
- How subscription messages work and that user authorization is required.
- What admins can see in the PC console.
- How users deactivate, retain, or delete data.
