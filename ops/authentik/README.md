# Draw Meatbags Authentik operations

Production authentication is provided by Authentik at `https://auth.meatbags.ru`.
Draw uses the OIDC issuer `https://auth.meatbags.ru/application/o/draw-meatbags/`
and the callback `https://draw.meatbags.ru/auth/callback`.

## Access policy

- Anyone may draw locally in their browser.
- Anonymous encrypted share links remain enabled and are limited to 5 MiB, 20
  creations per client per hour, and 100 creations globally per hour.
- Reaching an anonymous creation limit sends one Telegram operations alert and
  suppresses repeats for one hour.
- Persistent server canvases require an Authentik account in the `Draw users`
  group.
- New accounts can only be created with a single-use invitation for the
  `draw-invitation-enrollment` flow.
- The owner API token can list, read, and update the configured owner's canvases;
  it cannot delete them.

## Managed configuration

- `invitation-enrollment.yaml` defines the invite-only registration flow and
  assigns new users to `Draw users`.
- `admin-recovery.yaml` defines an admin-link-only password recovery flow. A
  direct visit without a generated recovery token is rejected.

Apply either blueprint from the Authentik server container with:

```bash
ak apply_blueprint /blueprints/custom/<filename>.yaml
```

Do not commit OIDC secrets, invitation tokens, recovery links, or API tokens.

## Invitation handoff

Create a single-use invitation in Authentik for the
`draw-invitation-enrollment` flow. Give the generated URL only to the intended
person. Opening the URL consumes the invitation session, so generate a new one
if the recipient abandons registration.

## Owner identity and rollback

OIDC canvas owners are stored as `oidc:<provider-subject>`. Before changing the
OIDC provider's subject mode or replacing a user, back up SQLite and migrate the
existing `canvases.user_id` values in one transaction.

The pre-OIDC production backup made on 2026-09-04 is stored outside the Docker
volume under `/var/backups/draw-meatbags/`. A rollback requires restoring that
database copy and removing the four `OIDC_*` variables from the Draw application;
the existing GitHub variables are intentionally retained for this purpose.
