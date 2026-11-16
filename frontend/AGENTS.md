# etheyl project rules

Shared API and security rules: [root AGENTS.md](../AGENTS.md).

## Workflow

- SvelteKit configuration belongs in `vite.config.ts`. Imports use the native `#lib` alias and explicit file extensions (`#lib/api/client.js`, `#lib/components/ui/button/index.js`).
- Use the installed shadcn-svelte components; read their barrels and official docs before use. Forms use `Field.FieldGroup` and `Field.Field`, callouts `Alert`, loading `Spinner`, empty states `Empty`.
- Do not import `src/lib/api/server.ts` in client code. Go owns authentication and business access; client visibility is not authorization.
- Do not manually edit `src/lib/api/schema.d.ts`; generate it with `pnpm generate:api`. The generator uses a separate TypeScript workspace; see [compatibility constraints](../docs/versions.md).
- Preserve HttpOnly sessions; do not store auth in localStorage. Remove action-token fragments before further navigation. Forms use `method="POST"`; do not submit passwords in query strings.
- Auth flows and resend placement: [README](../README.md#запуск). Match browser nickname `pattern` constraints with the server; escape literal hyphens for the HTML pattern `v` flag.
- Text button variants use `px-4`, including buttons with icons. Input focus/invalid rings are 1px; see [design rules](../DESIGN.md). Card footers use `mt-auto` so equal-height cards align.

## Svelte tools

- Load [svelte-code-writer](.agents/skills/svelte-code-writer/SKILL.md) and [svelte-core-bestpractices](.agents/skills/svelte-core-bestpractices/SKILL.md) when working with Svelte.
- For Svelte/SvelteKit tasks, call `list-sections` first, inspect `use_cases`, then fetch all relevant sections with `get-documentation`. Run `svelte-autofixer` on changed Svelte files until no issues or suggestions remain.
- Start Codex from `frontend/` in a trusted checkout to load [.codex/config.toml](.codex/config.toml); see [MCP setup](../README.md#svelte-mcp-и-skills). CLI commands, from this directory:

```sh
pnpm exec svelte-mcp list-sections
pnpm exec svelte-mcp get-documentation '<paths>'
pnpm exec svelte-mcp svelte-autofixer <file> --svelte-version 5.57.3
```

- For snippets outside project files, ask before generating a `playground-link`; do not use it for code written to project files.

## Commands

From `frontend/`; setup and local dev settings: [README](README.md).

```sh
pnpm install --frozen-lockfile
pnpm peers check
pnpm check:mcp
pnpm generate:api
pnpm check
pnpm build
pnpm format:check
pnpm test:e2e
```

E2E requires a running Compose stack and Chromium; see [test setup](README.md). Run desktop and mobile projects; override `E2E_BASE_URL`/`MAILPIT_URL` if needed. Do not bypass production rate limits for browser tests; use a fresh test stack for repeated registration flows.

## Billing (stage 4)

- Checkout lives at `/authors/[nickname]/subscribe?tier=ID`; account routes are `/account/subscriptions`, `/account/payments`, `/account/payments/[id]`. Shared billing UI lives in `src/lib/components/billing` and formatters in `src/lib/billing`.
- Use installed shadcn `Checkbox` for explicit consent (unchecked initially) and `NativeSelect` for level changes. Show exact RUB amounts, future charge terms and the test-shop label. Keep theme tokens centralized.
- A return from YooKassa never proves success. Payment GET is read-only; poll the server result, cap polling, clean up timers. Never grant access from query parameters or client state.
- Confirm current price in checkout/level-change payloads. Cancellation preserves paid access. `review` blocks new payment attempts; tell users the result is being checked.
- Run `pnpm test:e2e:billing` on a fresh Compose stack for desktop/mobile (the checkout failure response is mocked; backend lifecycle uses the integration fake provider). `pnpm build` enforces separate budgets for core route JS and the lazy analytics chart; see the stage 5 rules.

## Feed, discussion and analytics (stage 5)

- `/account/feed` uses opaque cursors from `/me/feed`, never offsets or client-filtered permission checks. Above-level posts show only public teaser/upgrade links. Paid access survives canceled renewal until expiry.
- `PostDiscussion` is keyed by post ID. Use the typed API for idempotent likes and plain-text flat comments; never render comment HTML. Clear displayed discussion if the API denies access. Preserve drafts after errors and disable duplicate submits.
- `/account/analytics` shows inclusive UTC date filters, confirmed test payments and a separately labeled current subscription snapshot. The current tier price must never replace the paid amount. `AnalyticsReport` loads `PaymentsChart` only in the browser; use the installed shadcn Chart/Table/Skeleton components and theme tokens. Keep the daily table available when the chart cannot load.
- LayerChart is pinned at `2.10.1`; use its SVG entry. Build checks count every asset: core JS ≤160 KB gzip, lazy chart JS ≤180 KB gzip, fonts ≤150 KB, CSS ≤20 KB. The manifest check rejects eager inclusion of the chart. Do not import chart components into global layouts.
- `pnpm test:e2e:social` requires a fresh Compose project ending in `-social-tests`, set through `E2E_SOCIAL_PROJECT`. Its fixture seeds only that test database; never add an HTTP grant endpoint. Run desktop/mobile; see [docs/social.md](../docs/social.md) for ports and commands.
