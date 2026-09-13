---
status: ready
priority: p1
issue_id: "001"
tags: [authentik, security, deployment]
dependencies: []
---

# Shared entry password, separate personal Draw storage

## Problem Statement
User approved an Authentik gate for all Draw access, with one shared guest credential and independent personal OIDC login for private server canvases.

## Findings
- Draw is served by OVH Coolify/Traefik under draw.meatbags.ru and board.meatbags.ru.
- OIDC application draw-meatbags is restricted to Draw users. Preserve this binding.
- Bearer Authorization is used by the app API: do not replace/intercept it with BasicAuth.
- PWA fallback must exclude outpost callbacks as well as /auth and /api.

## Proposed Solutions
- Approved: Authentik forward-auth gate with restricted shared guest, existing owner OIDC unchanged except explicit account selection/login.
- Rejected: global BasicAuth conflicts with app Authorization; per-person invitations and guest quotas exceed user scope.

## Recommended Action
Back up live state; configure separate gate application and restricted shared user; test outpost before cutover; deploy callback/login compatibility; protect both hostnames without restarting shared proxy; verify and retain rollback.

## Acceptance Criteria
- [ ] Backups and rollback documented.
- [ ] Anonymous root, snapshot API, storage API and live routes blocked on both hostnames.
- [ ] Shared credential opens local editor and live collaboration, not private server storage or account administration.
- [ ] Personal login remains available after shared gate login and private canvases load.
- [ ] Cached-browser callbacks reach Authentik, not PWA fallback.
- [ ] Existing canvases/snapshots preserved and relevant tests/build pass.
- [ ] Credential stored safely and user handoff completed.

## Work Log
### 2026-09-13 — Approved implementation
- Read live Authentik and Traefik configuration; no guest user or proxy provider currently exists.
- Implementation on codex/draw-shared-gate branches; no direct commit to production default branch.

## Resources
- https://docs.goauthentik.io/add-secure-apps/providers/proxy/server_traefik/
