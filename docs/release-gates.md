# Release Gates

1. Run backend tests: `go test ./...`
2. Run backend build: `go build -o study_plan_backend .`
3. Run the verified mini-program release build: `cd frontend && npm run release:mp-weixin`
4. Run admin type-check/build when `admin/` exists: `npm run type-check` and `npm run build`
5. Run OpenSpec validation for active changes: `openspec validate <change> --type change --strict --json --no-interactive`
6. Verify `https://slls.asia/health`, then complete `docs/wechat-submission-checklist.md` before WeChat submission.

Use `scripts/verify.ps1` for the default quality gate, or pass change names explicitly: `scripts/verify.ps1 quality-and-testing`.
