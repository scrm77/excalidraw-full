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
- [x] Backups and rollback documented.
- [x] Anonymous root, snapshot API, storage API and live routes blocked on both hostnames.
- [ ] Shared credential opens local editor and live collaboration, not private server storage or account administration.
- [ ] Personal login remains available after shared gate login and private canvases load.
- [ ] Cached-browser callbacks reach Authentik, not PWA fallback.
- [x] Existing canvases/snapshots preserved and relevant tests/build pass.
- [ ] Credential stored safely and user handoff completed.

## Work Log
### 2026-09-13 — Approved implementation
- Read live Authentik and Traefik configuration; no guest user or proxy provider currently exists.
- Implementation on codex/draw-shared-gate branches; no direct commit to production default branch.

### 2026-09-13 — Deployed; credential handoff pending
- Merged frontend PR scrm77/excalidraw#1 and parent PR #3; Coolify deployment f36g7wby6otlf33rbsjphmol finished at 80cad06f710b4308a8bb52746992c963824ab170.
- Gate installed through Traefik file provider, no proxy restart. Anonymous root/snapshot/live requests redirect; invalid private API Bearer returns 401; owner automation token still works.
- Recovery-established guest sessions (not password login) open editor. Two independent browsers joined one live room; text entered in the first appeared in the second. Guest private catalogue fetch returned 401. Shared account denied personal OIDC application and account-configuration flows.
- Guest Login displays the personal identity form. Existing personal app session listed nine canvases; a temporary canvas was created, read back via API, and survived reload. Removed only that exact test canvas; nine original canvas payloads match pre-change SHA-256 f50e67d6c9ba48432e2f073c99424c3571c2274cc963a690738393b7fc1295b8, SQLite integrity ok. Test browser storage preference restored.
- Go tests, frontend typecheck/lint/build passed. Live service worker contains all three callback exclusions; guest outpost callback traversed successfully.
- Remaining: owner sets draw-guest password in open Authentik admin form; verify actual shared-password login and full guest-to-personal password/callback journey. Password still unset at 07:48 UTC; no credential values recorded. Do not mark full acceptance complete from recovery-session tests.

## Resources
- https://docs.goauthentik.io/add-secure-apps/providers/proxy/server_traefik/
