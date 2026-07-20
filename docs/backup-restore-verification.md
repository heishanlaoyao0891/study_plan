# Backup And Restore Verification

1. Stop the backend process before copying SQLite files.
2. Copy `study_plan.db`, plus WAL/SHM files if present, to a timestamped backup directory.
3. Start a temporary backend with `DB_PATH` pointing at the copied database.
4. Run `go test ./...` against the temporary backend workspace when schema changes are involved.
5. Manually confirm login, plan list, today's tasks, check-ins, slack records, and admin overview load from the restored database.
6. For MySQL archive sync, verify row counts for `users`, `plans`, `daily_tasks`, `checkins`, `study_sessions`, and `slack_records` after sync.
