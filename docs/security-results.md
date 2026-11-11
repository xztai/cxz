# Результаты security-проверок после REVIEW.md

Проверены локально собранные образы. Grype 0.121.3 использует актуальную на момент запуска vulnerability DB; результаты зависят от её обновлений. Это инвентаризационный scan, он не доказывает достижимость каждого пути эксплуатации из etheyl.

Backend обновлён до zlib=1.4-r0. Frontend использует тот же исправленный zlib; ненужный npm с его brace-expansion/undici удалён из runtime. MinIO сохранён по явному решению владельца вместе с блокирующим security gate; его Go-зависимости в рамках исправления REVIEW.md не обновлялись.

| Образ | Critical | High | Medium | Low | Результат порога High |
| --- | ---: | ---: | ---: | ---: | --- |
| backend | 0 | 0 | 3 | 0 | PASS |
| frontend | 0 | 0 | 3 | 0 | PASS |
| minio | 9 | 35 | 14 | 1 | FAIL |

Количество — совпадения package/advisory (включая алиасы GHSA/GO/CVE), не число независимо доказанных эксплойтов. Неподавленные Medium у Alpine busybox остаются видимыми в отчётах.

## Блокирующие зависимости MinIO

Полный список доступен в Grype artifact CI. Ниже сгруппированы High/Critical по пакету; фиксированные версии — рекомендации базы сканера, совместимость с исходниками этого MinIO ещё требует отдельной проверки.

| Модуль | В образе | Исправленные версии из advisory |
| --- | --- | --- |
| `github.com/apache/thrift` | v0.21.0 | 0.23.0, 0.24.0, 0.25.0 |
| `github.com/go-jose/go-jose/v4` | v4.1.0 | 4.1.4, 4.1.6 |
| `github.com/klauspost/compress` | v1.18.0 | 1.20.2 |
| `github.com/prometheus/prometheus` | v0.303.0 | 0.305.2, 0.311.3 |
| `github.com/rabbitmq/amqp091-go` | v1.10.0 | 1.13.0 |
| `go.opentelemetry.io/otel/sdk` | v1.35.0 | 1.40.0 |
| `golang.org/x/crypto` | v0.37.0 | 0.52.0, 0.55.0, 0.56.0 |
| `golang.org/x/net` | v0.39.0 | 0.53.0, 0.55.0, 0.60.0 |
| `golang.org/x/text` | v0.24.0 | 0.39.0 |
| `google.golang.org/grpc` | v1.72.0 | 1.79.3, 1.82.1, 1.82.2, 1.83.1 |

High/Critical не добавлены в ignore/allowlist; CI завершает MinIO scan с ненулевым кодом. Публичный запуск хранилища с реальными данными остаётся заблокирован до отдельного исправления и проверки S3/совместимости. Последний upstream release проверен через [официальный GitHub](https://github.com/minio/minio/releases/tag/RELEASE.2025-10-15T17-29-55Z).

## Остальные проверки

- govulncheck 1.9.0: backend — PASS, уязвимостей не найдено.
- pnpm audit: High/Critical нет; один Low deepmerge, [GHSA-gcqx-w48m-w2h4](https://github.com/advisories/GHSA-gcqx-w48m-w2h4), исправленной версии нет. Advisory не подавлен.
- Gitleaks 8.30.1: доступная полная Git history и текущий tracked/non-ignored source — PASS. Игнорируемый локальный `.env` не входит в source scan; если он попадёт в Git, history/source scan его проверит.
- Пакет zlib=1.4-r0 соответствует исправлению [Alpine CVE-2026-4738](https://security.alpinelinux.org/vuln/CVE-2026-4738).
- Расписание, ответственность и сроки разбора описаны в [operations.md](operations.md).
