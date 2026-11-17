# Закреплённые версии и совместимость

Версии ниже взяты из файлов проекта. При обновлении меняйте manifest и lockfile
вместе, проверяйте peer dependencies, генерацию, типы и production-сборку.

| Часть                        | Версия / источник                                |
| ---------------------------- | ------------------------------------------------ |
| Go                           | `1.27.4`, `backend/go.mod`                       |
| sqlc                         | `1.31.1`, `make generate`                        |
| goose                        | `3.30.0`, `backend/go.mod`                       |
| golangci-lint                | `2.15.0`, `make lint`                            |
| Node.js                      | `26.x`, точная версия в `frontend/.node-version` |
| pnpm                         | `12.18.0`, `frontend/package.json`               |
| SvelteKit / Svelte           | `3.2.0` / `5.57.3`                               |
| TypeScript                   | `6.0.3` в приложении                             |
| shadcn-svelte / bits-ui      | `1.8.0` / `2.21.0`                               |
| LayerChart                   | `2.10.1`, добавлен на этапе 5                    |
| PostgreSQL и сервисы Compose | Точные теги и digest в `compose.yaml`            |

## Аналитика

shadcn-svelte Chart использует `layerchart@2.10.1`, совместимый с текущими
Svelte 5.57.3 и TypeScript 6.0.3. Используется SVG entry и динамический импорт
на экране аналитики. Основные маршруты не импортируют библиотеку графиков.
Native `#lib` aliases и явные `.js` extension сохраняются в скопированных UI
компонентах. Остальные версии зависимостей этапом 5 не меняются.

Проверки: `pnpm install --frozen-lockfile`, `pnpm peers check`, `pnpm check:mcp`,
`pnpm generate:api`, `pnpm check`, `pnpm build`, `make lint`, `make test-integration`.
