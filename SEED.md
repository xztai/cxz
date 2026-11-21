# Сид для локальной разработки etheyl

В запущенном Compose откройте [etheyl](http://localhost:8080). Общий пароль всех сидовых аккаунтов — `qweqweqweqweqwe`. Все адреса вымышлены. Локальная часть email совпадает с ником, домен — `@seed.etheyl.test`.

## Запуск

```sh
make init && make up                           # если Compose ещё не настроен или не запущен
SEED_ALLOW=1 make seed SEED_FLAGS=--dry-run     # предпросмотр текущего состояния без записи

# Если у локального воркера настроен тестовый ЮKassa, отключите его биллинг.
YOOKASSA_SHOP_ID= YOOKASSA_API_KEY= docker compose up -d --no-deps --wait worker

SEED_ALLOW=1 make seed                         # первый запуск
SEED_ALLOW=1 make seed                         # существующий сид сохраняется
SEED_ALLOW=1 make seed SEED_FLAGS=--reset       # восстановить сценарии относительно сегодняшнего дня
SEED_ALLOW=1 make seed SEED_FLAGS='--reset --dry-run'
```

`make seed` использует запущенные Postgres и RustFS из Compose через временные loopback-подключения. Требуются Python 3 и Go версии из `backend/go.mod`. При наличии `ffmpeg` с `libx264` добавляются два MP4-вложения. Сидер сообщит о пропуске видео, если `ffmpeg` отсутствует или не может создать MP4.

При заданном `DATABASE_URL` команда использует настройки текущего окружения. Для отдельного локального стека можно также запустить `go run ./cmd/seed` из `backend/` с теми же флагами. Передайте `SEED_ALLOW=1`, локальный `DATABASE_URL`, `APP_ORIGIN`, `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, при необходимости `S3_BUCKET` и `S3_SECURE=false` для HTTP. Для записи отключите обработку оплат на всех воркерах этого стека и задайте `SEED_BILLING_WORKER_DISABLED=1`. База должна быть мигрирована. Для `--reset` нужна роль владельца таблиц.

Синтетические `queued` и `pending` отложены на `infinity`. Автопродления остаются видимыми в API и могут стать просроченными со временем. Оставляйте биллинг воркера отключённым, пока эти данные есть в базе. Повторите команду перезапуска воркера после обычного `make up`, если Compose восстановит ключи из окружения или `.env`. API может оставаться с тестовыми ключами для проверки валидации checkout. У синтетических оплат нет ссылки на страницу провайдера.

`--reset` удаляет аккаунты зарезервированного домена и их данные. При подписках между сидовыми и обычными аккаунтами или реакциях обычных читателей на сидовые посты команда прервётся и откатится. Для удаления использованных уровней защитный триггер временно отключается внутри транзакции на время одной выборочной операции удаления. Внешние ключи продолжают действовать. После фиксации удаляются устаревшие объекты из префиксов `posts/seed-etheyl-v1/` и `profiles/seed-etheyl-v1/`. Идентификаторы и отметки времени обновятся, состав сценариев сохранится.

## Аккаунты и сценарии

| Email | Ник | Роль и сценарий |
|---|---|---|
| seed-admin@seed.etheyl.test | seed-admin | Администратор без профиля автора |
| seed-unverified@seed.etheyl.test | seed-unverified | Вход возвращает 403 до подтверждения email |
| seed-alina-verbova@seed.etheyl.test | seed-alina-verbova | Алина Вербова. Полный профиль, 4 уровня, 54 опубликованных поста, 2 черновика, медиа, аналитика за 90 дней |
| seed-lev-stone@seed.etheyl.test | seed-lev-stone | Lev Stone. Профиль без аватара и обложки, уровни 150,50 ₽ и 500 ₽, аналитика примерно за месяц |
| seed-nina-lis@seed.etheyl.test | seed-nina-lis | Нина Лис. Публичные публикации, уровней нет |
| seed-ira-park@seed.etheyl.test | seed-ira-park | Ирина Парк. Уровни есть, постов и подписчиков нет. «Пробный обжиг» за 0,50 ₽ возвращает `invalid_price` при checkout с тестовым биллингом |
| seed-mark-volkov@seed.etheyl.test | seed-mark-volkov | Марк Волков. Два архивных уровня. «Звуковая мастерская» использована, соседняя «Монтажная» свободна и дешевле |
| seed-reader@seed.etheyl.test | seed-reader | Лиза Невская. Читатель без профиля, подписок и платежей. Лента пуста |
| seed-daria-sokol@seed.etheyl.test | seed-daria-sokol | 1. Серебро до +20 дней с автопродлением, три последовательных платежа, смешанная лента |
| seed-mikhail-nord@seed.etheyl.test | seed-mikhail-nord | 2. Золото до +5 дней, автопродление выключено, смешанная лента |
| seed-olga-river@seed.etheyl.test | seed-olga-river | 3. Автопродление отменено пользователем, Серебро доступно до +10 дней |
| seed-pavel-gromov@seed.etheyl.test | seed-pavel-gromov | 4. Серебро истекло вчера. Три своих комментария на закрытом посте «Переулки за рынком» |
| seed-sonya-lee@seed.etheyl.test | seed-sonya-lee | 5. Серебро с переходом на Золото (799 ₽) в следующем периоде, смешанная лента |
| seed-anton-reed@seed.etheyl.test | seed-anton-reed | 6. Оплаченный доступ к архивному уровню Марка «Клуб ранних слушателей», причина остановки `tier_archived` |
| seed-vika-klen@seed.etheyl.test | seed-vika-klen | 7. `pending` у Алины, создан за 2 минуты до сида, и `queued` у Льва |
| seed-alex-roman@seed.etheyl.test | seed-alex-roman | 8. `review` с причиной `unknown_result` |
| seed-lena-zimina@seed.etheyl.test | seed-lena-zimina | 9. Три отменённых платежа с причинами `provider_rejected` и `payment_canceled` |
| seed-tim-krest@seed.etheyl.test | seed-tim-krest | 10. Подписки на Алину, Льва и Марка |
| seed-katya-dawn@seed.etheyl.test | seed-katya-dawn | 11. Один отменённый платёж и соглашение без оплаченного доступа |
| seed-denis-orlov@seed.etheyl.test | seed-denis-orlov | 12. История из 64+ оплат и доступ на Золото. Страницы по 24 |
| seed-maria-bel@seed.etheyl.test | seed-maria-bel | 13. Доступ к обоим уровням Льва |
| seed-kostya-more@seed.etheyl.test | seed-kostya-more | 14. Доступ к использованному активному уровню Марка и двум архивным уровням ниже по позиции |
| seed-vera-nik@seed.etheyl.test | seed-vera-nik | 15. Платина, доступ ко всем постам Алины |

У каждого из 30 дополнительных авторов есть публичная публикация и черновик для студии. Профилей хватает для второй страницы каталога. Покупки этих аккаунтов заполняют аналитику Алины и Льва.

| Email | Ник | Автор и тема |
|---|---|---|
| seed-anna-shum@seed.etheyl.test | seed-anna-shum | Анна Шум, камерные концерты |
| seed-gleb-sad@seed.etheyl.test | seed-gleb-sad | Глеб Садов, домашние сады |
| seed-maya-luna@seed.etheyl.test | seed-maya-luna | Maya Luna, иллюстрации к сказкам |
| seed-artem-yar@seed.etheyl.test | seed-artem-yar | Артём Яров, локальная история |
| seed-lili-fox@seed.etheyl.test | seed-lili-fox | Lili Fox, маленькие издательства |
| seed-sasha-bereg@seed.etheyl.test | seed-sasha-bereg | Александра Берег, морские маршруты |
| seed-roman-hill@seed.etheyl.test | seed-roman-hill | Roman Hill, урбанистика |
| seed-zoya-kim@seed.etheyl.test | seed-zoya-kim | Зоя Ким, бумажная пластика |
| seed-yuri-platon@seed.etheyl.test | seed-yuri-platon | Юрий Платонов, архитектурная графика |
| seed-nadia-sneg@seed.etheyl.test | seed-nadia-sneg | Надежда Снегирёва, сезонные рецепты |
| seed-marta-light@seed.etheyl.test | seed-marta-light | Marta Light, свет в интерьере |
| seed-boris-nov@seed.etheyl.test | seed-boris-nov | Борис Новиков, история музыки |
| seed-polina-vet@seed.etheyl.test | seed-polina-vet | Полина Ветрова, флористика |
| seed-val-shore@seed.etheyl.test | seed-val-shore | Val Shore, документальное кино |
| seed-sima-krai@seed.etheyl.test | seed-sima-krai | Серафима Край, текстиль |
| seed-ilya-sin@seed.etheyl.test | seed-ilya-sin | Илья Синицын, городские сады |
| seed-rita-bloc@seed.etheyl.test | seed-rita-bloc | Маргарита Блок, редактура и проза |
| seed-kir-tuman@seed.etheyl.test | seed-kir-tuman | Кирилл Туманов, пейзажная съёмка |
| seed-nora-quill@seed.etheyl.test | seed-nora-quill | Nora Quill, работа с пером |
| seed-ivan-bor@seed.etheyl.test | seed-ivan-bor | Иван Боров, пешие маршруты |
| seed-alla-luch@seed.etheyl.test | seed-alla-luch | Алла Луч, мозаика |
| seed-jonas-wood@seed.etheyl.test | seed-jonas-wood | Jonas Wood, резьба по дереву |
| seed-yana-kot@seed.etheyl.test | seed-yana-kot | Яна Котельникова, кошачьи наблюдения |
| seed-seva-mesh@seed.etheyl.test | seed-seva-mesh | Всеволод Мещеряков, модели и макеты |
| seed-mila-rain@seed.etheyl.test | seed-mila-rain | Mila Rain, поэзия |
| seed-oleg-barin@seed.etheyl.test | seed-oleg-barin | Олег Баринов, столярные соединения |
| seed-tanya-echo@seed.etheyl.test | seed-tanya-echo | Татьяна Эхо, радиотеатр |
| seed-fedor-len@seed.etheyl.test | seed-fedor-len | Фёдор Лёнин, лён и ткачество |
| seed-rosa-varga@seed.etheyl.test | seed-rosa-varga | Rosa Varga, полевые зарисовки |
| seed-vadim-kedr@seed.etheyl.test | seed-vadim-kedr | Вадим Кедров, экологические заметки |

## Где смотреть пограничные случаи

- **«7 маршрутов для утренней съёмки»** у Алины. Публичный пост с семью файлами при наличии FFmpeg, оригиналом на 50 МиБ для `Range`, миниатюрами, Markdown и длинной веткой обсуждения. Сервер использует CommonMark. Синтаксис Markdown-таблицы остаётся текстом, сырой HTML и схема `javascript:` обезвреживаются при рендеринге. В приложении есть лимит 100 МиБ на файл, число файлов на пост не ограничено.
- **«Город после дождя 🌧️»** и **«Трамвайное кольцо: дневник звука»**. Два пустых тизера, набор картинок и WAV. У **«Серебряного отпечатка»** тизер на 2000 символов и тело на 5000, у **«Откуда в кадре блики»** тело на 99 850 символов.
- **«Переулки за рынком»**. У Павла истёк оплаченный доступ, а его старые комментарии сохранены. Читатели с действующим Серебром увидят их в обсуждении. Сервер разрешает Павлу удалить собственный комментарий по ID, обсуждение для него закрыто.
- **Бронза Алины** стоит 0 ₽, но закрытые посты по-прежнему требуют подходящий оплаченный уровень. Минимум checkout равен 100 копейкам, поэтому отдельного гранта на Бронзу нет.
- **«Первый сезон: записи у старой плотины»** и **«Архив: ночной вокзал»** показывают, как архивные позиции продолжают участвовать в проверке доступа. Антон видит первый пост, Константин видит оба. «Монтажная» оставлена свободной для проверки ограничений перетаскивания. У неё нет поста.
- **Оплаты** хранят валюту RUB в JSON заявки, а API выставляет `test: true`. Отдельных полей для валюты и `test` в таблице нет. Для читателя с одним отменённым платежом интерфейс подписок может показать соглашение без оплаченного периода.
