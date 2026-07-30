## Context

The repository currently contains a working `slls.asia` TLS proxy, a production frontend environment that points to the same origin, a Compose stack with an additional IP test port, and several older guides that still use placeholder domains. The WeChat manifest has the real AppID, but there is no single release command that proves the production API origin and generates a reviewed artifact.

The public H5 application, `/admin/`, and `/api/` can all share one origin. This is simpler than introducing separate API and admin subdomains and requires only one WeChat legal request domain.

## Goals / Non-Goals

**Goals:**

- Establish `https://slls.asia` as the canonical production origin.
- Eliminate direct public IP access from the normal production stack.
- Make mini-program release builds deterministic and fail early on unsafe configuration.
- Produce a clear operator handoff from DNS/TLS verification through WeChat upload and rollback readiness.

**Non-Goals:**

- Automate WeChat account login, code upload, review approval, or publication.
- Change application features, authentication contracts, or backend persistence.
- Move the admin console or assets to additional subdomains in this release.

## Decisions

### Use one HTTPS origin

H5 is served at `/`, admin at `/admin/`, health at `/health`, and API requests at `/api/` under `https://slls.asia`. This matches the current nested Nginx proxies and minimizes DNS, certificate, CORS, and WeChat-domain configuration. Separate subdomains remain a future scaling option.

### Validate release inputs before building

A repository script reads `frontend/.env.production` and `frontend/src/manifest.json`, rejects missing HTTPS, IP-literal origins, path-bearing origins, enabled development login override, placeholder AppID, and non-incremented version metadata, then runs frontend tests, type checking, and the WeChat production build. It also inspects generated files to ensure the canonical origin and AppID are present.

### Keep external publication manual

The release command stops at a verified `frontend/dist/build/mp-weixin` artifact. Upload and review require an authenticated WeChat developer identity, version notes, category/privacy declarations, and platform approval, so those steps are documented with explicit evidence fields rather than hidden behind an unreliable automation.

### Remove the IP test port from production Compose

The frontend container remains bound to loopback for the host Nginx/container gateway, while the extra public `:82` mapping is removed. Emergency diagnostics use SSH plus loopback curl instead of bypassing TLS and the canonical domain.

## Risks / Trade-offs

- [Existing users retain an old API override] -> Production builds disable the override and remove stored override values at runtime.
- [Certificate or DNS is not ready when code deploys] -> The checklist gates publication on DNS, certificate-chain, HTTPS health, and API smoke checks.
- [A release reuses a version number] -> Release validation requires numeric version metadata and this change increments the manifest to `1.0.1` / `101`; subsequent releases must increment it again.
- [Same-origin admin is more discoverable] -> The admin remains authentication-protected and is not linked from the mini program; a separate admin domain can be introduced later.
- [Automated build passes while WeChat review fails] -> External console settings, privacy declarations, category qualification, and real-device checks remain explicit manual gates.

## Migration Plan

1. Confirm `slls.asia` and `www.slls.asia` DNS resolve to the production server and the certificate chain is valid.
2. Deploy the updated Compose and Nginx configuration while keeping backend data volumes unchanged.
3. Verify H5, `/admin/`, `/health`, and representative `/api/` responses through HTTPS.
4. Add `https://slls.asia` to the WeChat request legal-domain list and confirm it is accepted.
5. Build the `1.0.1` mini-program artifact, run real-device smoke tests, upload it, and submit it for review.
6. Roll back application code through the previous Git revision if necessary; retain the domain/TLS proxy and persistent data volumes.

## Open Questions

- The final WeChat version description and review submission time remain operator choices.
- Avatar hosting currently references `assets.slls.asia`; its DNS/TLS readiness must be confirmed before enabling MinIO-backed public avatars.
