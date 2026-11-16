# etheyl — frontend

SvelteKit 3, Svelte 5, TypeScript, shadcn-svelte. Общая инструкция запуска — [README в корне](../README.md); правила разработки и MCP — [AGENTS.md](AGENTS.md).

```sh
pnpm install --frozen-lockfile
pnpm peers check
pnpm check:mcp
pnpm generate:api
pnpm check
pnpm build
```

Для E2E сначала запустите Compose из корня, затем:

```sh
pnpm exec playwright install chromium
pnpm test:e2e
```

Локальный dev-сервер: `pnpm dev --host 127.0.0.1`. API должен работать на `localhost:8081` с `APP_ORIGIN=http://localhost:5173`, `COOKIE_SECURE=false`, `TRUST_PROXY=false`. В Docker используется adapter-node, а API доступен браузеру через общий reverse proxy.

Тема и типографика централизованы в `src/routes/layout.css`. OpenAPI-клиент генерируется из `../docs/openapi.yaml`; сгенерированные типы не редактируйте вручную.

Публикации: `/account/posts` (редактор `/account/posts/<id>`), чтение `/posts/<id>`.
Поддерживаются Markdown, изображения, оригинальные аудио/видео и вложения до 100 МБ.
`pnpm test:e2e:publications` выполняет отдельный набор на desktop/mobile; используйте
свежий отдельный Compose-стек и `E2E_BASE_URL`/`MAILPIT_URL`, как описано в корневом
README. Наборы auth и publications используют обычные лимиты и запускаются в разных
окружениях. Минимальный уровень проверяет Go, клиент получает только разрешённые поля.

## Лента и аналитика

Лента: `/account/feed`; аналитика: `/account/analytics`; лайки и комментарии — под
опубликованным постом. Для браузерной группы этапа 5 запускайте
`pnpm test:e2e:social` с `E2E_SOCIAL_PROJECT` и выделенным свежим Compose.
Полная команда: [docs/social.md](../docs/social.md#проверки).
График LayerChart 2.10.1 загружается отдельно; JS-бюджеты разделены на основной
интерфейс (160 KB gzip) и ленивый график (180 KB gzip), общие лимиты CSS/шрифтов
сохранены. API-типы генерируются через `pnpm generate:api`.
