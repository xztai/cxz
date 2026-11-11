You are able to use the Svelte MCP server, where you have access to comprehensive Svelte 5 and SvelteKit documentation. Here's how to use the available tools effectively:

## Available Svelte MCP Tools:

### 1. list-sections

Use this FIRST to discover all available documentation sections. Returns a structured list with titles, use_cases, and paths.
When asked about Svelte or SvelteKit topics, ALWAYS use this tool at the start of the chat to find relevant sections.

### 2. get-documentation

Retrieves full documentation content for specific sections. Accepts single or multiple sections.
After calling the list-sections tool, you MUST analyze the returned documentation sections (especially the use_cases field) and then use the get-documentation tool to fetch ALL documentation sections that are relevant for the user's task.

### 3. svelte-autofixer

Analyzes Svelte code and returns issues and suggestions.
You MUST use this tool whenever writing Svelte code before sending it to the user. Keep calling it until no issues or suggestions are returned.

### 4. playground-link

Generates a Svelte Playground link with the provided code.
After completing the code, ask the user if they want a playground link. Only call this tool after user confirmation and NEVER if code was written to files in their project.

## etheyl project rules

- The scaffold and `pnpx sv add ai-tools` generated the instructions above. Keep them when updating this file.
- SvelteKit 3.2.0 configuration belongs in `vite.config.ts`, not `svelte.config.js`. Use Svelte 5 runes and TypeScript 6.0.3. Imports use the native `#lib` alias and explicit file extensions (`#lib/api/client.js`, `#lib/components/ui/button/index.js`).
- Use shadcn-svelte (nova) components from `src/lib/components/ui/`. Compose forms with `Field.FieldGroup` and `Field.Field`, callouts with `Alert`, loading with `Spinner`/`Skeleton`, empty states with `Empty`. Read the installed components' barrels and official docs before use.
- `src/routes`: screens/loaders; `src/lib/components`: composed UI; `src/lib/api/client.ts`: stateless browser API with CSRF middleware; `src/lib/api/server.ts`: server-only API (never import in client code). Go owns authentication and business access; client visibility is not authorization.
- `docs/openapi.yaml` at repository root is the API contract. Do not manually edit `src/lib/api/schema.d.ts`. Generate via `pnpm generate:api`. The generator lives in `tools/openapi` with isolated TypeScript 5.9.3; the app uses TypeScript 6. `pnpm-workspace.yaml` pins runed 0.38.0 for SvelteKit 3 compatibility.
- Preserve HttpOnly sessions; never store auth in localStorage. Action tokens arrive in URL fragments and are removed before further navigation. Forms use `method="POST"` and never submit passwords in query strings. Never log tokens or request bodies.
- Before email confirmation, fetch the address from `/auth/verify-email/preview`; do not trust an email embedded in a link. Confirming creates the backend session and navigates to `/account` with invalidated load data. Duplicate registration remains on the form and offers login, without suggesting resend.
- Registration collects a unique `nickname`, not a personal name: 3–30 ASCII letters/digits/`_`/`-`, no dots or spaces. The server lowercases it; `User.nickname` is the canonical account/profile URL identifier. Login remains email-based. Match browser `pattern` constraints with the server (escape literal hyphens for the HTML pattern `v` flag).
- Show email-confirmation resend only inside the «Проверьте почту» card, after registration or a correct-password login for an unverified account. No resend links on login, registration errors or the verification page. `/auth/resend` redirects old bookmarks to login. A rejected reset to the current password keeps the form and token available for a different password.
- Text button variants use `px-4`, including buttons with icons; avoid icon-specific padding reductions. Input focus/invalid rings are 1px. Card footers use `mt-auto` so equal-height cards align their footers.
- MCP is configured in `.codex/config.toml` in this directory, using Node and pinned local `@sveltejs/mcp` installed by pnpm. Start Codex from `frontend/` in a trusted checkout so it loads that project configuration. A running older agent session may need reopening to discover the added tools/skills. Offline CLI equivalent: `pnpm exec svelte-mcp`.
- Official skills are installed under `.agents/skills/svelte-code-writer/` and `.agents/skills/svelte-core-bestpractices/`; load their SKILL.md when working with Svelte. They will be available to the agent on its next turn/session. Use `pnpm exec svelte-mcp list-sections`, then `get-documentation '<paths>'`. Before completion, run `pnpm exec svelte-mcp svelte-autofixer <file> --svelte-version 5.57.3` on changed Svelte files and resolve reported issues.
- Commands (from this directory): `pnpm install --frozen-lockfile`, `pnpm peers check`, `pnpm check:mcp`, `pnpm generate:api`, `pnpm check`, `pnpm build`, `pnpm format:check`, `pnpm test:e2e`. Local dev: `pnpm dev --host 127.0.0.1`, with Go on `localhost:8081`, `APP_ORIGIN=http://localhost:5173`, `COOKIE_SECURE=false` and `TRUST_PROXY=false`. Node is pinned by `.node-version`; pnpm by `packageManager`.
- E2E tests use an already running Compose at `http://localhost:8080`, Mailpit at `http://localhost:8025`; override `E2E_BASE_URL`/`MAILPIT_URL` if needed. Install Chromium via `pnpm exec playwright install chromium`. Run desktop and mobile projects. Do not bypass production rate limits for browser tests; use a fresh test stack when repeatedly running registration flows.
