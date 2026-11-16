# Backend etheyl

Shared API and security rules: [root AGENTS.md](../AGENTS.md).

## Workflow

- Before editing Go, run the installed Modern Go Guidelines CLI and follow its version-specific guidance:

```sh
sh <skill-dir>/scripts/run-tool.sh list --file-path <file>
# Portable pinned equivalent, from backend/:
go run github.com/JetBrains/go-modern-guidelines@v0.1.1 list --go-version 1.27
```

- Do not edit generated `internal/store` files manually. Change `queries/*.sql` and run `make generate` from the repository root.
- Do not change an already deployed migration. See [release and migration constraints](../docs/operations.md#релиз-и-миграции).
- Business mutations spanning users, tokens, sessions and mail use a single transaction with `store.New(tx)`. Lock the user before consuming a token or creating a session; roll back on failure. Set the session cookie only after commit.
- Return only stable Russian messages and API error codes to clients.
- For reset to the current password, compare under the user row lock. Consume/check the token before comparison so concurrent replays return `invalid_token`; rollback preserves the token and all sessions on `password_unchanged`.
- Auth behavior and storage guarantees: [README](../README.md#авторизация-и-ограничения).

## Checks

Before completion, from `backend/`:

```sh
gofmt -w cmd internal
go test ./...
go vet ./...
```

From the repository root, run `make lint`; for auth changes also run `make test-integration` and `make e2e` with Compose. Integration tests require `TEST_DATABASE_URL` and a dedicated database ending in `_test`; they apply migrations and truncate its tables.

## Billing (stage 4)

- Business rules live in `internal/billing`; HTTP DTOs/routes in `internal/httpapi/billing.go`; queries in `queries/billing.sql`. `subscriptions` contains paid grants only, never checkout intent.
- Use `YOOKASSA_API_KEY` and `YOOKASSA_SHOP_ID` together; MVP rejects live key prefixes and non-test provider payments. No frontend secrets or provider payload logging.
- Persist immutable payment request and idempotence key before network I/O. Never create a replacement for an unresolved attempt. Unknown IDs stop at 23h; known IDs use GET. Lock author → agreement → payment; never hold a transaction across provider HTTP. Terminal transitions and paid grants/mail commit together.
- Browser mutations require Origin + CSRF. The separate server-to-server webhook rejects cookies/origin, verifies via authenticated provider GET, and never trusts event amounts/status/metadata. See [billing operations](../docs/billing.md).
- Cancellation/archival stop unsent renewals; already authorized sends must reconcile. Never retry a declined charge automatically. Preserve frozen prices, original calendar day and paid access.
- Run `make test-integration` for every billing change (includes race tests), `make lint`, `make generate`; `make test-restore` covers payment tables. Manual reconciliation: `docker compose exec api etheyl reconcile-payment --provider-id ID` (GET only).

## Feed, discussion and analytics (stage 5)

- `queries/social.sql` owns feed/comments/likes, `queries/analytics.sql` owns report queries; handlers are `internal/httpapi/social.go` and `analytics.go`. Migration `00007_social.sql` defines the shared `post_readable` predicate for text/media/feed/interactions. Keep this predicate aligned when adding moderation.
- Feed uses active paid grants and `(published_at,id)` cursors, 24 rows. Never return closed bodies/media via summaries. Discussion requires an authenticated user with published-post access. Likes are idempotent; comments are plain text, 1–2000 Unicode characters and 20 creates/user/15min. Owner-only comment deletion remains available after grant expiry without revealing text.
- Social writes lock user → post, check access and commit atomically. Do not acquire creator locks after a post lock. Reads of discussion and analytics use a read-only repeatable-read snapshot.
- Analytics is always scoped to the session's author. Dates are inclusive UTC days, 1–366; sum immutable `payments.amount_kopecks` only for succeeded attempts with provider ID and `completed_at` in the period. Active subscriptions are a current snapshot, including canceled renewals/archived levels with paid access. Exclude blocked users and future/expired/revoked grants.
- Run `make test-integration`, `make lint`, `make generate` and `make test-restore`; social integration tests cover authorization, paging, concurrent likes, duplicate payments and UTC boundaries. [Stage 5 operations/tests](../docs/social.md).
