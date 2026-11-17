# etheyl landing — shape brief

Status: confirmed by the product owner on 2026-11-17. The conventional direction
and Boosty/Patreon reference bar are confirmed. This document is a plan, not an
implementation contract.

## Job and audience

Target `/`, implemented later in `frontend/src/routes/+page.svelte`. Mode:
Persuade. Lead with artists, 3D modelers, and other creators evaluating a place
to publish work and receive support. Supporter discovery is secondary.
Use Russian copy for the Russian target region, on desktop and mobile.

## Outcome and evidence

Primary action: register now, with `Создать аккаунт`. Explain the mechanism in
plain language: publish work and offer access by subscription. Success means
the visitor understands this relationship and can begin registration without
having to search for the action. Email verification follows registration.

Use original illustrative artwork and a creator-page demonstration explicitly
labeled `Демонстрационный пример`. Include illustration and 3D work. No real
creator assets, testimonials, audience counts, or income evidence were supplied.

## Selected direction

The user chose the familiar creator-platform direction, keeping comp-first.
Its visual reference is `.impeccable/mocks/decision/canon.png`: a light ground,
strong dark type, blue actions, and artwork presented inside a creator-page
example. This is a selected direction preview, not final build-comp approval.

User-named craft references: [Boosty](https://boosty.to/) and
[Patreon](https://www.patreon.com/), public pages reviewed 2026-11-17.
Boosty makes starting a creator page and the setup sequence explicit; Patreon
foregrounds creators and their work alongside membership and community.
Use their familiarity and product clarity as the bar. etheyl retains its own
name and original artwork; their capabilities and commercial claims do not
establish etheyl's capabilities.

## Sequence and interaction

1. Header: etheyl, `Как это работает`, `Найти авторов`, `Войти`.
2. Opening: `Создавайте. Делитесь. Получайте поддержку.` and
   `Публикуйте работы и открывайте доступ по подписке.` Registration sits beside
   a large, labeled creator-page example. The focal moment is seeing artwork
   in the place the visitor could create for their own audience.
3. Setup sequence: create and verify an account, set up a creator page, publish
   work and configure subscription access. Keep account creation distinct from
   a published creator page.
4. Concrete examples: show how a public introduction and subscription content
   relate, using illustration and 3D-work examples. Explain the mechanism with
   content, rather than a generic feature-card grid or invented results.
5. Close: concise answers to verified setup/access questions, then a repeated
   invitation that returns focus to the existing registration form. Footer
   destinations must exist before shipping.

No carousel or elaborate motion is needed. On mobile, preserve the offer and
registration priority; stack the demonstration and explanatory content at
readable sizes. Use restrained feedback for focus, submission, and errors.

## States and content ranges

- Three registration fields: email, nickname, password; visible labels and an
  accessible password-visibility control. Accept the API's limits, including
  nickname length 3–30 and password minimum 12 Unicode characters, maximum
  256 UTF-8 bytes. Do not silently trim or normalize passwords.
- Design empty, focused, invalid, submitting, duplicate email/nickname,
  request failure, rate-limit, and check-email success states. Preserve safe
  input on recoverable failure and announce feedback accessibly.
- One illustrative creator-page preview with at least one illustration and
  one 3D-work example. Keep it visibly demonstrative; no invented prices or
  real-person identity. Support multiline Russian headings without clipping.
- Keyboard navigation, visible focus, readable contrast, reduced motion, and
  ordinary browser zoom remain baseline quality requirements.

## Scope and open decisions

Shape delivers this landing plan and direction previews only. Implementation
will need to connect registration to the existing API and provide working
login, verification, and creator-discovery destinations in the reset frontend.
Those dependencies do not authorize redesigning the account, billing, or
administration screens as part of this landing.

Keep handlers, generated types, and OpenAPI aligned. Preserve exact Origin and
cookie/header CSRF protections, including registration/login, and the shared
rules against logging credentials, tokens, cookies, or request bodies.

Differentiation, fees, payout timing, supported payment methods, licensing,
and detailed moderation promises remain unconfirmed. Do not fill the landing
with assumptions about them. Final artwork, build composition, and production
readiness of linked flows require resolution during implementation.
