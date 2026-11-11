# etheyl

Первый этап MVP сервиса авторских подписок: рабочая авторизация, русский интерфейс, кабинеты аккаунта и администратора, инфраструктура и CI. Один аккаунт может использоваться автором и подписчиком. Профили, уровни, публикации, платежи и модерация относятся к следующим этапам.

## Запуск

Нужны Docker Engine с Compose v2+ и Python 3 для генерации локальных секретов. Для проверок вне Docker — Go 1.27.4, Node 26.13.0 и pnpm 12.18.0. Версии и причины выбора описаны в [docs/versions.md](docs/versions.md).

```sh
make init
make up
```

`make init` создаёт `.env` со случайными паролями ролей PostgreSQL и MinIO, ключом шифрования outbox и правами `0600`. В существующий файл дописывает только отсутствующие настройки, сохраняя прежние значения. Первая сборка MinIO занимает несколько минут. `make up` собирает образы, выполняет goose-миграции и ждёт готовности сервисов.

- Приложение: **http://localhost:8080**.
- Локальная почта: **http://localhost:8025** (Mailpit).
- Готовность API: `GET http://localhost:8080/api/v1/health`.

Регистрируйтесь через интерфейс с уникальным никнеймом: 3–30 символов, латинские буквы, цифры, `_` и `-`, без точек и пробелов. Никнеймы сохраняются в нижнем регистре; `Author` и `author` считаются одним никнеймом. Он предназначен для будущего URL профиля автора. Вход выполняется по email.

Откройте письмо в Mailpit и подтвердите email кнопкой на странице из письма. Страница показывает подтверждаемый адрес; после подтверждения автоматически создаётся сессия и открывается личный кабинет. Повторная регистрация занятого email или никнейма сразу возвращает ошибку и не отправляет новое письмо. Кнопка повторной отправки подтверждения находится только в карточке «Проверьте почту»: после регистрации или попытки входа с правильным паролем в неподтверждённый аккаунт. Для подтверждённых аккаунтов она не показывается.

Восстановление пароля открывается со страницы входа или из личного кабинета. Текущий пароль нельзя установить повторно: API вернёт ошибку, сохранив ссылку для следующей попытки и действующие сессии. Письма обычно приходят в течение секунды; при временном сбое SMTP worker повторяет отправку с задержкой.

Миграция `00003_secure_outbox.sql` удаляет ожидающие письма старого plaintext-формата: после обновления запросите такие письма заново. Уже доставленные ссылки продолжают действовать.

Миграция `00002_nicknames.sql` назначает ранее созданным аккаунтам уникальные никнеймы `user-<id>`. Их email, пароли, статусы и сессии сохраняются; прежнее имя остаётся в БД для обратимости миграции.

Открывайте именно `localhost:8080`: Origin запросов должен совпадать с `APP_ORIGIN`. При смене адреса меняйте `.env` и пересоздавайте сервисы. Конфигурация Compose предназначена для локальной разработки: HTTP, почтовый перехватчик, порты привязаны к loopback. Порядок перехода к HTTPS, внешнему SMTP, backup/restore и эксплуатации описан в [docs/operations.md](docs/operations.md).

```sh
docker compose ps
docker compose logs -f api worker
make down
```

Остановка сохраняет volumes `postgres_data` и `minio_data`. PostgreSQL, API и MinIO не публикуют порты на хост; консоль MinIO отключена. Buckets по умолчанию приватны; создание bucket и работа с медиа появятся на этапе 2.

## Первый администратор

После запуска выполните в терминале:

```sh
docker compose exec api etheyl admin --email admin@example.com --nickname admin
```

CLI запросит пароль без отображения (минимум 12 символов), создаст подтверждённый аккаунт администратора. Если email или никнейм уже занят, команда завершится ошибкой; существующие аккаунты она не повышает. Затем войдите через обычную форму по email и откройте `/admin`. Доступ проверяет Go API. Публичная регистрация не принимает признак администратора.

Для автоматизации есть `--password-stdin`: пароль передавайте через stdin, не через аргументы командной строки. Миграции можно безопасно повторить: `docker compose run --rm migrate`.

## Структура

```text
backend/
  cmd/etheyl/             API, worker, migrate, admin, healthcheck
  internal/accounts/     транзакционные сценарии регистрации, входа, verify/reset
  internal/auth/         Argon2id, случайные токены, нормализация email
  internal/httpapi/      REST /api/v1 и интеграционные тесты
  internal/mailqueue/    SMTP и PostgreSQL outbox
  internal/config/       конфигурация окружения
  internal/observability/ безопасная диагностика ошибок
  internal/store/        сгенерированный sqlc-код
  queries/               исходные SQL-запросы
  migrations/            SQL goose, встроенные в бинарный файл
frontend/
  src/routes/            SvelteKit-экраны и серверные проверки сессии
  src/lib/components/    UI, включая shadcn-svelte
  src/lib/api/           типизированный клиент и генерируемые типы
  tools/openapi/         изолированный генератор OpenAPI
  tests/e2e/             Playwright: desktop и mobile
  .agents/skills/        официальные Svelte skills
  .codex/config.toml     локальный Svelte MCP для Codex
docs/openapi.yaml        контракт API
infra/                   reverse proxy и сборка MinIO
compose.yaml             локальное окружение
compose.test.yaml        отдельная временная БД тестов
```

Инструкции для разработки: [frontend/AGENTS.md](frontend/AGENTS.md) и [backend/AGENTS.md](backend/AGENTS.md).

## Проверки

```sh
cd frontend
pnpm install --frozen-lockfile
pnpm peers check
pnpm test:unit
pnpm check
pnpm build
pnpm generate:api
cd ..
make generate
make lint
make test
make test-integration
make test-restore
```

`make test-integration` запускает отдельный PostgreSQL на `127.0.0.1:5433`, применяет миграции и выполняет Go-тесты с `-race`. Интеграционные тесты разрешены только на БД с суффиксом `_test` и очищают её таблицы. Обычный `make test` выполняет unit-тесты; интеграционные пропускаются без `TEST_DATABASE_URL`.

С запущенным приложением:

```sh
cd frontend
pnpm exec playwright install chromium
pnpm test:e2e
```

Playwright проверяет интерфейс регистрации, доставку письма worker через Mailpit, подтверждение, ошибочный и успешный вход, SSR после перезагрузки, восстановление, отзыв старой сессии, запрет `/admin`, выход, адаптивность и клавиатурную навигацию. Использует уникальные тестовые email и оставляет эти аккаунты в локальной БД. Повторные прогоны учитываются обычными лимитами регистрации; используйте отдельное чистое тестовое окружение или дождитесь окончания 15-минутного окна.

CI выполняет sqlc-генерацию с проверкой изменений, Go-тесты с PostgreSQL и `-race`, golangci-lint, проверку сгенерированных API-типов, peer-зависимостей, Svelte/TypeScript, production-сборку, чистый запуск Compose и Playwright на двух размерах экрана. GitHub Actions закреплены по SHA. Security workflow по PR/push и расписанию проверяет Go/npm, секреты и три итоговых образа, сохраняет отчёты и CycloneDX. Дополнительные проверки: права runtime-ролей и реальный pg_dump/pg_restore. Подробности исправления аудита — [REVIEW.md](REVIEW.md), эксплуатация — [docs/operations.md](docs/operations.md). Локальный отчёт первого этапа — [docs/stage-1.md](docs/stage-1.md).

## Svelte MCP и skills

Установщик запускался из `frontend` командой **`pnpx sv add ai-tools`**. Сгенерированный `AGENTS.md` сохранён и дополнен правилами проекта. Для клиента Other/Codex добавлена локальная конфигурация; сервер `@sveltejs/mcp` закреплён в lockfile, официальные skills установлены через skill-installer в `frontend/.agents/skills`.

Запускайте Codex из `frontend/` в доверенном checkout, чтобы он подхватил `frontend/.codex/config.toml`. Новые skills доступны на следующем ходе/в новой сессии; для обнаружения нового MCP уже запущенному клиенту может понадобиться перезапуск.

```sh
cd frontend
pnpm check:mcp
pnpm exec svelte-mcp list-sections
pnpm exec svelte-mcp get-documentation 'kit/routing,svelte/$state'
pnpm exec svelte-mcp svelte-autofixer src/lib/components/AuthForm.svelte --svelte-version 5.57.3
```

Документация: [Svelte CLI](https://svelte.dev/docs/cli/overview), [ai-tools](https://svelte.dev/docs/cli/ai-tools), [Svelte AI](https://svelte.dev/docs/ai/overview).

## Авторизация и ограничения

Пароли: Argon2id, 64 MiB, 3 прохода, 2 потока, случайная соль. В таблицах сессий и action-токенов хранятся SHA-256-хеши случайных 256-битных значений. Сессия действует 30 дней, подтверждение email — 24 часа, сброс пароля — 1 час. Восстановление отзывает все сессии; выход — текущую. Cookie HttpOnly, SameSite=Strict, host-only; для HTTPS обязательны Secure и префикс `__Host-`.

Каждый POST требует точный Origin и совпадение CSRF-header с HttpOnly-cookie, включая регистрацию и вход. БД хранит общие для экземпляров API лимиты: 60 POST/IP за 15 минут, 5 регистраций/IP, 10 неудачных входов на пару IP/email, 3 запроса письма на пару IP/email/тип. Успех очищает счётчик пары; повышенный риск email добавляет ограниченную задержку, сохраняя вход с другого IP. Глобальный бюджет 3 письма/email/тип молча объединяет лишние запросы и сохраняет последнюю ссылку. Cookie и токены не попадают в журналы; ссылки в письмах используют fragment и не отправляют токен в URL HTTP-запроса. Пароли не обрезаются и не нормализуются.

Каждый запрос ограничен deadline 5 с; SQL имеет statement_timeout 4 с и lock_timeout 2 с. Лимит четырёх вычислений Argon2 не удерживается при ожидании SQL или чтении тела. Каждый браузерный fetch имеет собственный таймаут 10 с.

Пользователь, токен и письмо создаются в одной транзакции. Тело outbox зашифровано AES-256-GCM, ключи вынесены в `OUTBOX_KEYS` отдельно от БД. API не может читать тела очереди; worker имеет только необходимые права, мигратор — отдельную роль владельца схемы. Worker выбирает письма через `FOR UPDATE SKIP LOCKED`; успешные удаляет, неуспешные откладывает, максимум 8 попыток, затем явное состояние failed. Актуальное письмо можно повторить через `docker compose exec worker etheyl requeue-mail --id 123`; просроченные и использованные ссылки не отправляются. Внутренний `/api/v1/mail/status` worker показывает backlog/failed/heartbeat, отдельно от liveness. SMTP имеет семантику at-least-once: после аварии письмо может повториться, но токен останется одноразовым. Завершённые сессии, просроченные токены/лимиты и старые записи outbox очищаются worker. Время в БД — `timestamptz`, инфраструктура работает в UTC.

Настройки backend: `DATABASE_URL`, `OUTBOX_KEYS`, `REQUEST_TIMEOUT`, `APP_ORIGIN`, `HTTP_ADDR`, `COOKIE_SECURE`, `TRUST_PROXY`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_INSECURE`. По умолчанию SMTP требует TLS; Compose явно включает локальный режим. `TRUST_PROXY=true` допустим за доверенным reverse proxy: Caddy перезаписывает `X-Real-IP`, а API не опубликован наружу. В ручном dev-запуске оставляйте `TRUST_PROXY=false`.

Шрифты поставляются только в WOFF2 для Latin/Cyrillic; `pnpm build` проверяет бюджет шрифтов 150 КБ, JavaScript 120 КБ gzip и CSS 20 КБ gzip для всей client-поставки.
