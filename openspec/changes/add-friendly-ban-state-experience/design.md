## Context

The API normally uses `{code, message, data}`, but `api.Fail` sends HTTP 200. Ban checks were duplicated and returned different shapes, while clients only handled a subset of HTTP 403 responses. `BanUser` stores permanent bans with a 2099 sentinel, which clients must not infer independently.

## Goals / Non-Goals

**Goals:** return one privacy-safe real-403 envelope, clear expired bans centrally, preserve tokens during active bans, support tokenless login bans, align countdowns to server time, and prevent routing loops.

**Non-Goals:** expose a public user lookup, permit feedback APIs while banned, replace the persisted ban schema, or revoke JWTs solely because a ban is active.

## Decisions

### Canonical backend state evaluation

A shared backend package evaluates `banned_until` once. Active bans return HTTP 403 and code 403 with `data.account_banned`, `reason`, RFC3339 `banned_until`, `permanent`, and RFC3339 `server_now`. It returns no user identity or profile fields. Expired bans clear both persisted ban fields and allow the caller to continue.

### Server-owned permanent semantics

The model owns construction and recognition of the 2099 sentinel. Responses expose `permanent` explicitly; clients never guess permanence from the date.

### Existing `/auth/me` as refresh mechanism

No new endpoint is needed. Middleware returns the canonical ban response while active and clears an expired ban before `/auth/me` proceeds. The retained JWT therefore supports safe recovery without broadening unauthenticated account discovery.

### Central client interception

The request wrapper recognizes both HTTP 403 and legacy/business code 403 only when `account_banned` is true. It validates and bounds the safe payload, stores whether a token exists, retains that token, and re-launches the banned page unless already there or routing is in progress.

### Countdown and recovery

The page computes server time as response `server_now` plus monotonic wall-clock elapsed since receipt. Foreground events reconcile immediately and hidden/unloaded pages clear intervals. A retained-token session calls `/auth/me` on foreground/manual action and at timed expiry; success clears ban state and uses `routeForUser`. Tokenless login bans return to login instead of pretending they can refresh authentication.

## Risks / Trade-offs

- Wall-clock changes after receipt can still affect elapsed time; each server response resets the offset and foreground refresh limits drift.
- A user can navigate before making a request, but launch state and every protected API response route back immediately.
- Feedback remains authenticated and blocked, so the page provides safe administrator support text rather than an unusable link.

## Security Review

- The payload contains only reason and timing/status metadata.
- Tokens are retained only locally and never placed in route parameters or ban storage.
- Tokenless responses cannot query ban status by username or user ID.
- WeChat linking checks active state before mutating OpenID association.
