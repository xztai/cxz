# etheyl

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

- Artists, 3D modelers, and other creators who want to monetize their work.
- People who want to support those creators.

These audiences and motivations are confirmed by the product owner. Specific
creator disciplines beyond these examples, experience levels, and usage contexts
remain open.

## Product Purpose

Help creators monetize their work and enable people to support them. Future
product decisions should serve both sides of that relationship.

The intended creator-subscription service includes payments, subscriptions,
a feed, comments, creator analytics, and moderation at launch. Detailed feature
behavior and measures of success remain open.

## Positioning

The intended audience is confirmed; differentiation from other creator-support
and subscription services is undecided. Do not invent competitive advantages,
income promises, fees, or creator outcomes.

## Operating Context

The confirmed target region is Russia.

Repository evidence, rather than a confirmed launch specification:

- The README describes one account that can act as both creator and subscriber.
- The API contract includes creator profiles and discovery, subscription tiers,
  posts, media, checkout, payment history, and subscription management.
- The frontend is a SvelteKit/TypeScript web project and is currently being
  rebuilt from a minimal scaffold. Historical README workflows are not proof
  that corresponding screens are available in the current frontend.

## Capabilities and Constraints

Confirmed launch scope: payments, subscriptions, feed, comments, creator
analytics, and moderation. This is intended scope, not a claim that these
features are already implemented. README descriptions conflict on implementation
status; verify behavior before making availability claims.

The product owner specified no additional product or accessibility requirements.

Preserve the shared rules in `AGENTS.md`: keep handlers, generated frontend types,
and `docs/openapi.yaml` aligned; validate exact Origin and cookie/header CSRF on
unsafe requests, including login; trust proxy headers only when explicitly
configured; never log passwords, cookies, email links, tokens, or request bodies.

Open product decisions:

- Primary interface language, currency, and payment availability within the
  confirmed Russian target region. Russian repository documentation and a
  YooKassa API endpoint are evidence of prior implementation choices.
- Supported creator deliverables and whether 3D work needs special workflows.
  A 3D-modeler audience alone does not establish a marketplace or model viewer.
- Supported devices.

## Brand Commitments

The existing product name is `etheyl`. No additional voice, asset, or visual
commitments have been confirmed for the rebuilt frontend.

## Evidence on Hand

- `docs/openapi.yaml`: API contract, including auth, profiles, tiers, posts,
  media, billing, and subscriptions; not evidence of launch readiness.
- `README.md`: historical product and operational context; contains conflicting
  scope statements and references to removed frontend features and documents.
- `frontend/package.json`: current frontend stack and development commands.
- `docs/operations.md`: operational documentation.

Testimonials, customer counts, revenue figures, and other marketing proof have
not been supplied or confirmed. Do not fabricate them.

## Product Principles

- Keep creator monetization and supporter participation central to decisions.
- Treat artists and 3D modelers as explicit audiences when evaluating workflows.
- Describe capabilities and outcomes only as far as verified evidence supports.
