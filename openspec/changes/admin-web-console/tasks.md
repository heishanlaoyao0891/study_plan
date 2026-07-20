## 1. Mini Program Cleanup

- [x] 1.1 Remove admin page route from mini program pages configuration
- [x] 1.2 Remove admin entry button from mini program plan page
- [x] 1.3 Remove unused mini program admin API calls if no longer needed by user-facing pages
- [x] 1.4 Confirm role/user/config management is not reachable from mini program UI

## 2. Admin Console Setup

- [x] 2.1 Create PC admin Vue 3 + TypeScript + Vite frontend project or workspace
- [x] 2.2 Configure admin API base URL and token storage
- [x] 2.3 Implement username/password admin login and session handling
- [x] 2.4 Add route guard requiring admin role
- [x] 2.5 Document recommended separate admin domain deployment

## 3. Admin Credential Backend

- [ ] 3.1 Add admin credential model or configured bootstrap admin
- [ ] 3.2 Store admin passwords using a secure password hash
- [ ] 3.3 Add admin username/password login endpoint
- [ ] 3.4 Add login failure handling and basic rate limiting
- [ ] 3.5 Bootstrap initial admin credentials from environment variables

## 4. Admin MVP Features

- [ ] 4.1 Implement overview dashboard with key counts
- [ ] 4.2 Implement user list with search/filter
- [ ] 4.3 Implement user detail summary with role/status visibility
- [ ] 4.4 Implement ban/unban with duration and reason
- [ ] 4.5 Implement global slack configuration editor
- [ ] 4.6 Implement per-user slack configuration editor
- [ ] 4.7 Implement AI model configuration editor
- [ ] 4.8 Implement WeChat subscription message configuration editor
- [ ] 4.9 Implement audit log list

## 5. Audit And Backend Support

- [ ] 5.1 Add admin audit log model
- [ ] 5.2 Record audit logs for login, ban/unban, and config changes
- [ ] 5.3 Add audit log list API for admin console
- [ ] 5.4 Add overview metrics API if existing APIs are insufficient
- [ ] 5.5 Add AI configuration model and admin APIs
- [ ] 5.6 Add subscription message configuration model and admin APIs
- [ ] 5.7 Encrypt stored AI API keys when server-side encryption secret is configured

## 6. Verification

- [ ] 6.1 Verify backend build
- [ ] 6.2 Verify mini program build after removing admin surface
- [ ] 6.3 Verify admin console build
- [ ] 6.4 Validate OpenSpec change
