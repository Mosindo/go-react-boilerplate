# Payments Policy

Lien is 100% free for its users. There is intentionally no payments agent, provider or code path.

## Constraints

- Do not add Stripe, in-app purchases, subscriptions, paid boosts, paywalled features or advertising SDKs.
- Do not gate any essential feature (likes, seeing who liked you, chatting, photos) behind a purchase.
- If sustainability work is ever needed, it needs an explicit product decision from the founder first.

## History

The former `go-react-saas` boilerplate shipped a Stripe billing module and subscription tables. They were removed (migration `014_drop_legacy_boilerplate.sql`).
