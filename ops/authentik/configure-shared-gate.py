"""Run with `ak shell` inside the existing Authentik server after a backup.

No passwords or tokens are printed. The shared password is set separately from
Bitwarden; reruns do not rotate credentials or change the owner's membership.
"""

from django.db import transaction
from authentik.core.models import Application, Group, User
from authentik.flows.models import Flow
from authentik.outposts.models import Outpost
from authentik.policies.models import PolicyBinding
from authentik.policies.expression.models import ExpressionPolicy
from authentik.providers.proxy.models import ProxyProvider

with transaction.atomic():
    owner_app = Application.objects.get(slug="draw-meatbags")
    owner_group = Group.objects.get(name="Draw users")
    assert owner_app.bindings.filter(group=owner_group).exists()

    guests, _ = Group.objects.get_or_create(name="Draw gate guests")
    assert not guests.is_superuser
    guest, created = User.objects.get_or_create(
        username="draw-guest",
        defaults={
            "name": "Draw shared guest",
            "type": "external",
            "path": "users/draw-shared-gate",
            "attributes": {"draw_shared_gate": True},
        },
    )
    assert guest.attributes.get("draw_shared_gate") is True
    assert not guest.is_superuser and not guest.ak_groups.filter(pk=owner_group.pk).exists()
    if created:
        guest.set_unusable_password()
    guest.attributes.update({
        "goauthentik.io/user/can-change-name": False,
        "goauthentik.io/user/can-change-email": False,
        "goauthentik.io/user/can-change-username": False,
    })
    guest.save()
    guest.ak_groups.add(guests)

    provider, _ = ProxyProvider.objects.update_or_create(
        name="Draw shared entry gate",
        defaults={
            "mode": "forward_single",
            "external_host": "https://draw.meatbags.ru",
            "authentication_flow": Flow.objects.get(slug="default-authentication-flow"),
            "authorization_flow": Flow.objects.get(slug="default-provider-authorization-implicit-consent"),
            "invalidation_flow": Flow.objects.get(slug="default-provider-invalidation-flow"),
            "intercept_header_auth": False,
            "basic_auth_enabled": False,
            "skip_path_regex": "",
            "access_token_validity": "minutes=10",
            "refresh_token_validity": "days=7",
        },
    )
    provider.set_oauth_defaults()
    provider.save()
    app, _ = Application.objects.update_or_create(
        slug="draw-shared-gate",
        defaults={
            "name": "Draw shared entry",
            "provider": provider,
            "policy_engine_mode": "any",
            "meta_launch_url": "https://draw.meatbags.ru/",
        },
    )
    for order, group in enumerate([guests, owner_group]):
        PolicyBinding.objects.get_or_create(target=app, group=group, order=order)
    outpost = Outpost.objects.get(managed="goauthentik.io/outposts/embedded")
    outpost.providers.add(provider)

    # Shared users must not replace the shared password or add their own MFA.
    # These default flows have no existing policy bindings; fail rather than
    # change an unrelated policy composition if that assumption stops being true.
    policy, _ = ExpressionPolicy.objects.update_or_create(
        name="draw-shared-guest-no-account-configuration",
        defaults={"expression": "return not request.user.attributes.get('draw_shared_gate', False)"},
    )
    for flow in Flow.objects.filter(designation="stage_configuration").exclude(slug="initial-setup"):
        assert not flow.bindings.exclude(policy=policy).exists(), flow.slug
        PolicyBinding.objects.get_or_create(target=flow, policy=policy, order=900)

print("GATE_CONFIGURED", {
    "username": guest.username,
    "password_set": guest.has_usable_password(),
    "gate_application": app.slug,
    "owner_application": owner_app.slug,
    "owner_membership": guest.ak_groups.filter(pk=owner_group.pk).exists(),
})
