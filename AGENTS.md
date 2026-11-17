# etheyl

## Shared rules

- Keep handlers, generated frontend types and the [API contract](docs/openapi.yaml) aligned.
- Require exact Origin and cookie/header CSRF validation for unsafe requests, including login. Trust proxy headers only when explicitly configured.
- Do not log passwords, cookies, email links, tokens or request bodies.

## Package rules

- [Backend](backend/AGENTS.md)
- [Frontend](frontend/AGENTS.md)
