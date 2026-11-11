# Закреплённые версии

Проверено 11 ноября 2026. Зависимости Go закреплены в `backend/go.mod`/`go.sum`, JavaScript — точными версиями в `frontend/package.json`, workspace-пакете генератора и `pnpm-lock.yaml`. Образы закреплены версиями и digest; локальные образы собираются из этих источников.

| Компонент | Версия |
| --- | --- |
| Go | 1.27.4 |
| pgx | 5.12.0 |
| sqlc | 1.31.1 |
| goose | 3.30.0 |
| golangci-lint | 2.15.0 |
| PostgreSQL | 18.6 |
| Node.js, официальный Docker-образ | 26.13.0 LTS |
| pnpm | 12.18.0 |
| SvelteKit / Svelte | 3.2.0 / 5.57.3 |
| adapter-node | 6.0.2 |
| Vite / Tailwind CSS | 8.3.7 / 4.3.5 |
| TypeScript приложения | 6.0.3 |
| shadcn-svelte / bits-ui / runed | 1.8.0 / 2.21.0 / 0.38.0 |
| openapi-typescript / TypeScript генератора | 7.13.0 / 5.9.3 |
| openapi-fetch | 0.17.0 |
| Svelte MCP | 0.1.30 |
| Playwright | 1.65.0 |
| Caddy / Mailpit | 2.11.9 / 1.33.1 |
| MinIO Community | RELEASE.2025-10-15T17-29-55Z |

Node 26.13.1 уже опубликован как дистрибутив, но официальный Docker-тег `node:26.13.1-alpine` во время реализации недоступен. Закреплён последний доступный совместимый официальный Alpine-образ 26.13.0; та же версия используется CI и `.node-version`.

SvelteKit 3 требует TypeScript 6, тогда как стабильный openapi-typescript декларирует TypeScript 5. Генератор размещён в отдельном workspace, поэтому оба используют поддерживаемые версии. У bits-ui диапазон runed относится к SvelteKit 2; override закрепляет стабильный runed 0.38.0 с поддержкой Kit 3. `pnpm peers check` проходит без подавления предупреждений.

Последний security-релиз MinIO Community доступен исходным кодом. `infra/minio.Dockerfile` собирает точный upstream-тег через Go module proxy с проверкой Go checksum database. Сервис использует постоянный volume и закрытую сеть; консоль отключена.

Источники версий: [Go](https://go.dev/dl/), [sqlc](https://github.com/sqlc-dev/sqlc/releases), [goose](https://github.com/pressly/goose/releases), [golangci-lint](https://github.com/golangci/golangci-lint/releases), [PostgreSQL](https://www.postgresql.org/support/versioning/), [Node LTS](https://nodejs.org/en/about/previous-releases), [MinIO](https://github.com/minio/minio/releases), [Caddy](https://github.com/caddyserver/caddy/releases), [Mailpit](https://github.com/axllent/mailpit/releases). Версии npm дополнительно сверены с registry через pnpm; Docker-теги — с registry и успешной сборкой.
