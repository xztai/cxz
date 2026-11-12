# Product — etheyl

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

- Primary: independent creators bringing an existing audience to the service.
- Secondary: supporters joining to follow and support those creators.
- One person can be both a creator and a subscriber using the same account.

Audience priority and the shared-account model were confirmed during initialization.

## Product Purpose

etheyl is a Russian-language creator subscription service being built in stages. Its purpose is to give independent creators a place to connect with their existing audience and receive support.

The implementation includes the account foundation, public creator profiles, a searchable creator catalog, image uploads, and subscription tier configuration. Creators can set up their page and tiers; visitors can find them through the catalog. Publishing, subscribing, and payments remain planned capabilities.

## Positioning

Open decision: the distinguishing mechanism or position relative to other creator subscription services is undecided. Do not invent competitive advantages or imply that fees, reach, earnings, or payment availability have been established.

## Operating Context

- Acquisition starts with creators bringing their existing audience. Discovery of unfamiliar creators is not the confirmed first priority.
- Registration collects an email, a unique nickname, and a password. Sign-in uses email, not the nickname.
- Email confirmation shows the address obtained from the server; successful confirmation signs the user in and opens `/account`.
- The account area has subscriber and creator modes. These are two uses of the same account, not separate account types.
- Password recovery uses email. A successful password reset revokes existing sessions.
- Administrators use the same sign-in flow and a server-protected `/admin` area. Administrator accounts are created through the CLI; public registration does not grant that role.
- The repository runs locally through Docker Compose at `http://localhost:8080`, with Mailpit for development email. This is development infrastructure, not evidence of a public deployment. Setup and operational details live in `README.md` and `docs/operations.md`.

## Capabilities and Constraints

The current scope and the following product commitments were confirmed during initialization; implementation details are grounded in the linked repository sources.

### Available now

- Public introductory page and Russian-language authentication screens.
- Registration, email confirmation, confirmation resend in the eligible flow, sign-in, sign-out, and password recovery/reset.
- Account identity and verification status, subscriber/creator mode switching, and an administrator access screen.
- Creator profiles with a unique nickname-based URL, public name, description, avatar and cover; first save publishes the profile in the searchable catalog.
- Owner-only editing of profiles and subscription tiers; tier prices are integer RUB kopecks, higher tiers include lower tiers, and new tiers are added at the top. Used tier positions are fixed; used tiers are archived instead of deleted.
- Validated JPEG/PNG/WebP uploads in private MinIO storage, with metadata in PostgreSQL and public delivery through the API.
- Server-enforced sessions and administrator access. Authentication state must not become a browser-only permission check.

### Preserve in future work

- Keep one account for creating and supporting others.
- A nickname is a unique account identifier used in the creator-profile URL, not a personal name. Changing it changes the public URL; the previous URL stops resolving. It accepts 3–30 ASCII letters, digits, underscores, or hyphens, is stored lowercase, and is unique without regard to case.
- Show confirmation resend only in the “Проверьте почту” state after registration or a correct-password sign-in for an unverified account. Duplicate registration is an error, not a resend flow.
- Reject resetting to the current password while preserving the recovery link for another attempt. Only a successful change revokes sessions.
- Preserve server-managed HttpOnly sessions and the existing security boundaries. Detailed engineering rules live in `frontend/AGENTS.md` and `backend/AGENTS.md`; the API contract is `docs/openapi.yaml`.
- Keep the distinction between working capabilities and planned stages explicit in product copy and empty states.

### Planned, not currently available

Publications, subscription purchases, payments, and moderation are not implemented. A protected administrator page does not mean moderation tools exist. See `docs/stage-1.md` for the recorded stage boundary.

### Open decisions

- Competitive positioning and differentiation.
- Launch geography; Russian interface language alone does not establish a launch market.
- Commissions, payment providers, and payout terms. Tier configuration currently uses RUB; payment availability is not established.
- Product-specific accessibility requirements or a formal conformance target.

## Brand Commitments

- Preserve the name `etheyl`.
- Preserve Russian as the interface language.
- No additional binding voice or visual requirements were established during initialization.

## Evidence on Hand

- `README.md`: product scope, working account flows, setup, and operational constraints.
- `docs/stage-1.md`: implementation boundaries and a historical report of local checks; not a claim that those checks were rerun during initialization.
- `frontend/src/routes/`: implemented home, authentication, account, and administrator screens, including copy that identifies future functionality.
- `frontend/src/lib/components/Logo.svelte`, `frontend/src/lib/components/WingMark.svelte`, and `frontend/src/lib/assets/favicon.svg`: existing identity assets.
- `frontend/tests/e2e/` and `frontend/playwright.config.ts`: browser scenarios for desktop and mobile, including account flows and keyboard interaction.
- No customer testimonials, adoption metrics, revenue results, payment coverage, or launch-market evidence was established in this initialization. Future work needs real evidence before making those claims.

## Product Principles

1. Prioritize creators arriving with their own audience and make joining understandable for their supporters.
2. Let people create and support others without maintaining separate accounts.
3. Make account access reliable and preserve the security and recovery guarantees already implemented.
4. Describe shipped functionality accurately and distinguish it from future subscription, publishing, and payment capabilities.
