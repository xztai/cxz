# etheyl frontend

SvelteKit 3 / Svelte 5, TypeScript and pnpm. The Russian landing at `/` follows
[the landing brief](../docs/landing-brief.md). Fonts and demonstration artwork
are served locally; artwork generation prompts are in
[landing-artwork.json](../docs/landing-artwork.json).

## Development

```sh
pnpm install
pnpm dev --host 127.0.0.1
```

Browser API calls use `/api/v1` on the same origin with cookies. The Vite dev
and preview servers proxy `/api` to `http://127.0.0.1:8081`; override that target
with `API_PROXY_TARGET` (see `.env.example`). Configure the backend's
`APP_ORIGIN` to the exact frontend URL, including protocol, hostname and port.
The proxy preserves the browser's Origin; it does not bypass backend validation.
In production, the existing Caddy configuration routes `/api` directly to Go.

Unsafe requests first obtain a fresh CSRF token and HttpOnly cookie from
`GET /api/v1/auth/csrf`, then submit JSON with `X-CSRF-Token`. Passwords are
neither trimmed nor normalized. Form values and auth tokens are never persisted
to browser storage or logged. Verification and password-reset pages read the
backend email's `#token=…` fragment and immediately remove it from the URL.
Verification previews the address and requires an explicit confirmation before
consuming the token.

## Routes

- `/`: landing, registration, check-email state and resend.
- `/login`: login and successful session confirmation.
- `/auth/verify`, `/auth/resend`: email verification and replacement links.
- `/auth/forgot`, `/auth/reset`: password recovery and reset.
- `/authors`: public author search and pagination, with empty/error states.
- `/authors/[nickname]`: public profile details.

Creator setup, publishing, checkout, account management and administration
screens are outside this landing implementation. Login and verification use the
backend's session cookie and offer author discovery after success.

## Validation

```sh
pnpm generate:api
pnpm check
pnpm exec vitest run --project server
pnpm exec playwright install chromium
pnpm exec playwright test
pnpm build
```

`generate:api` regenerates `src/lib/api/schema.d.ts` from `docs/openapi.yaml`.
Browser tests start the production preview and mock API responses. They cover
CSRF cookie/header transmission, exact password preservation, Unicode validation,
recoverable failures, verification confirmation, discovery and responsive layout.
They do not exercise a live database, SMTP delivery or payment flows. Production
hosting still needs an appropriate SvelteKit adapter; the scaffold retains
`adapter-auto`.
