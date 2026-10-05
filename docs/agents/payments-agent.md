# Payments Agent — intentionally disabled

Lueur is **free**: no subscription, no paywall, no paid boost, no artificial like limit, no mandatory ads.
There is therefore nothing for a payments agent to do, and the original Stripe billing module is not wired
into the API.

## Constraints

- Never reconnect `services/api/internal/features/billing` or add any gating of core features behind a payment.
- Rate limits are abuse guards only and must stay generous enough for normal use.
- If a voluntary donation or a non-gating revenue model is ever introduced, it needs an explicit product decision
  from the founder first and must not restrict likes, matches, messages or visibility.
