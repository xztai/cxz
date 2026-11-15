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
