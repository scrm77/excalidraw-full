# Draw shared-entry gate

Approved 2026-09-13 in https://github.com/scrm77/excalidraw-full/issues/2.

## Boundaries

- `draw-shared-gate` is a separate Authentik proxy application on the embedded outpost.
- `draw-guest` belongs only to `Draw gate guests`, not `Draw users`; no admin/RBAC privileges.
- Existing OIDC `draw-meatbags` remains restricted to `Draw users`.
- Shared password does not log users into the private Draw catalogue. Local browser drawings remain separate for each browser.
- Owner sets shared password under Authentik Admin → Directory → Users → draw-guest → Set password. Do not use the owner's own account/password for guests.
- Standard account-configuration flows deny the shared guest (password/profile/MFA); initial-setup keeps its existing built-in restriction.
- Snapshot and live storage behavior is unchanged. Gate controls access, not disk/RAM quotas or retention.

## Routing

Install `draw-shared-gate.traefik.yaml` as `/data/coolify/proxy/dynamic/draw-shared-gate.yaml` on OVH. File provider watches it; no shared proxy restart is needed. `board.meatbags.ru` redirects to the protected canonical host, preserving path/query (and browser fragments).

An explicit narrow exception preserves `/api/v2/kv[/...]` requests carrying Bearer credentials. These always pass through Draw's existing `AuthJWTOrOwnerAPI` middleware: invalid tokens fail 401; the owner automation token retains its existing no-delete restriction. Do not generalize this exception to `/api`, `/v1` or `/socket.io`.

`OIDC_LOGIN_FLOW_URL=https://auth.meatbags.ru/if/flow/default-authentication-flow/` makes the app Login button show an identity form before personal OIDC authorization. This avoids rejecting a guest SSO identity before they can switch to their personal account.

PWA navigation fallback excludes `/auth`, `/api`, and `/outpost.goauthentik.io`. Previously cached editor assets/local drawings can remain usable offline; the gate blocks new server access, not bytes already stored in browsers. Never clear user site data as part of deployment.

## Backup and rollback

Pre-change backup: `/var/backups/draw-meatbags/20260913T072530Z-pre-shared-gate/` on OVH, owner-only directory. Contains Authentik PostgreSQL dump, SQLite online backup (9 canvases, 8 snapshots), and Traefik configuration archive.

To immediately remove only the new gate, move `/data/coolify/proxy/dynamic/draw-shared-gate.yaml` into that backup directory with a `.disabled` suffix. Existing Coolify Docker routes become active again. This does not remove boards or users. Do not restore all shared proxy configuration or the entire Authentik database unless a separate, reviewed recovery requires it.

The optional login-flow env can be removed independently to restore the old direct OIDC redirect. The app/PWA changes are backward-compatible without the gate.

## Verification checklist

- No-cookie requests to root, snapshots and live endpoints redirect to Authentik. Alternate hostname redirects to canonical protected host.
- Invalid Bearer on private API returns 401; actual owner token still lists its own canvases.
- Guest gate session opens editor; cannot obtain personal OIDC login/token or access owner's catalogue; cannot change shared password or configure MFA.
- Login after guest session shows personal identity form and returns to Draw after owner authentication.
- Two guest clients can join the same live room and observe an edit.
- Existing SQLite data remains intact. No load test or unrelated account/email workflow changes.

## Observed deployment, 2026-09-13

Production commit `80cad06f710b4308a8bb52746992c963824ab170`, Coolify deployment `f36g7wby6otlf33rbsjphmol` finished. Anonymous route checks, guest private API 401, owner API, and live two-browser edit propagation passed. Test guest SSO sessions were established using short-lived recovery links, not a password.

Existing personal session loaded nine private canvases, created one temporary test canvas and retained it after reload. Only that test canvas was removed. Original nine canvas payloads match the backup digest; SQLite integrity is ok. Frontend build/typecheck/lint and Go tests passed.

Credential handoff remains pending: owner must set `draw-guest` password in the open Authentik admin form. Actual shared-password login and complete guest-to-owner password/callback login have not been verified. Guest Login currently reaches an empty identity form; existing owner app JWT server access is verified separately. Track remaining acceptance in issue #2; do not present recovery-session testing as password verification.
