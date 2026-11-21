package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/store"
)

const (
	alina = "alina-verbova"
	lev   = "lev-stone"
	nina  = "nina-lis"
	ira   = "ira-park"
	mark  = "mark-volkov"

	bronze   = "Бронза"
	silver   = "Серебро"
	gold     = "Золото"
	platinum = "Платина"
	levEntry = "Ближний круг"
	levPro   = "Открытая студия"
	archived = "Клуб ранних слушателей"
	retired  = "Полевые записи"
	markUsed = "Звуковая мастерская"
	markFree = "Монтажная"
)

type account struct {
	nick, name, topic string
}

func (a account) nickname() string { return "seed-" + a.nick }
func (a account) email() string    { return a.nickname() + emailSuffix }

var accounts = []account{
	{"admin", "Администратор etheyl", ""},
	{"unverified", "Яна Котова", ""},
	{alina, "Алина Вербова", "Городская фотография, визуальные дневники и прогулки с камерой"},
	{lev, "Lev Stone", "Рисунок тушью, смешивание пигментов и работа с бумагой"},
	{nina, "Нина Лис", "Библиотечные заметки, открытые эссе и книжный клуб"},
	{ira, "Ирина Парк", "Керамика и глазури. Готовлю первые материалы для студии"},
	{mark, "Марк Волков", "Полевые записи, звуковой монтаж и архив подкаста"},
	{"reader", "Лиза Невская", ""},
	{"daria-sokol", "Дарья Соколова", ""},
	{"mikhail-nord", "Михаил Норд", ""},
	{"olga-river", "Ольга Речная", ""},
	{"pavel-gromov", "Павел Громов", ""},
	{"sonya-lee", "Софья Ли", ""},
	{"anton-reed", "Антон Рид", ""},
	{"vika-klen", "Виктория Клён", ""},
	{"alex-roman", "Alex Roman", ""},
	{"lena-zimina", "Елена Зимина", ""},
	{"tim-krest", "Тимур Крестов", ""},
	{"katya-dawn", "Катя Рассветова", ""},
	{"denis-orlov", "Денис Орлов", ""},
	{"maria-bel", "Мария Белова", ""},
	{"kostya-more", "Константин Морев", ""},
	{"vera-nik", "Вера Никитина", ""},
	{"anna-shum", "Анна Шум", "камерные концерты"},
	{"gleb-sad", "Глеб Садов", "домашние сады"},
	{"maya-luna", "Maya Luna", "иллюстрации к сказкам"},
	{"artem-yar", "Артём Яров", "локальная история"},
	{"lili-fox", "Lili Fox", "маленькие издательства"},
	{"sasha-bereg", "Александра Берег", "морские маршруты"},
	{"roman-hill", "Roman Hill", "урбанистика"},
	{"zoya-kim", "Зоя Ким", "бумажная пластика"},
	{"yuri-platon", "Юрий Платонов", "архитектурная графика"},
	{"nadia-sneg", "Надежда Снегирёва", "сезонные рецепты"},
	{"marta-light", "Marta Light", "свет в интерьере"},
	{"boris-nov", "Борис Новиков", "история музыки"},
	{"polina-vet", "Полина Ветрова", "флористика"},
	{"val-shore", "Val Shore", "документальное кино"},
	{"sima-krai", "Серафима Край", "текстиль"},
	{"ilya-sin", "Илья Синицын", "городские сады"},
	{"rita-bloc", "Маргарита Блок", "редактура и проза"},
	{"kir-tuman", "Кирилл Туманов", "пейзажная съёмка"},
	{"nora-quill", "Nora Quill", "работа с пером"},
	{"ivan-bor", "Иван Боров", "пешие маршруты"},
	{"alla-luch", "Алла Луч", "мозаика"},
	{"jonas-wood", "Jonas Wood", "резьба по дереву"},
	{"yana-kot", "Яна Котельникова", "кошачьи наблюдения"},
	{"seva-mesh", "Всеволод Мещеряков", "модели и макеты"},
	{"mila-rain", "Mila Rain", "поэзия"},
	{"oleg-barin", "Олег Баринов", "столярные соединения"},
	{"tanya-echo", "Татьяна Эхо", "радиотеатр"},
	{"fedor-len", "Фёдор Лёнин", "лен и ткачество"},
	{"rosa-varga", "Rosa Varga", "полевые зарисовки"},
	{"vadim-kedr", "Вадим Кедров", "экологические заметки"},
}

var readers = accounts[8:23]
var catalog = accounts[23:]

var studioParagraphs = []string{
	"На рассвете улица наполняется отражениями. В витринах ещё горит свет, а на мостовой уже видно первое солнце. Я начинаю прогулку без маршрута и записываю, где меняется цвет.",
	"Хорошее упражнение для следующей прогулки — взять один объектив и повторять один и тот же ракурс с разной дистанции. Сравните поля кадра и то, как меняется ощущение пространства.",
	"Дома я раскладываю контактные листы на столе, отмечаю удачные переходы и записываю настройки рядом с кадрами. На следующий день передвигаю фотографии и проверяю, держится ли последовательность без объяснений.",
	"В архив попадают схемы маршрутов, фотографии вывесок и голоса дворов. Когда детали из разных источников совпадают, у заметки появляется место и время, к которым можно вернуться.",
	"Последний проход посвящаю цвету. Сравниваю экран с пробным отпечатком при дневном свете. В завершённой серии всегда оставляю техническую заметку про бумагу, экспозицию и ограничения.",
}

func sizedText(length int, paragraphs []string) string {
	text := strings.Join(paragraphs, "\n\n") + "\n\n"
	runes := []rune(strings.Repeat(text, length/len([]rune(text))+1))
	return string(runes[:length])
}

func (s *seeder) createAccounts() error {
	for i, a := range accounts {
		nick, ok := auth.NormalizeNickname(a.nickname())
		if !ok || nick != a.nickname() {
			return fmt.Errorf("неверный ник сидового аккаунта %d", i)
		}
		hash, err := auth.HashPassword(devPassword)
		if err != nil {
			return err
		}
		var id int64
		created := s.now.AddDate(0, -16, -i)
		err = s.tx.QueryRow(s.ctx, `
			INSERT INTO users (email, nickname, name, password_hash, email_verified, is_admin, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id
		`, a.email(), nick, a.name, hash, a.nick != "unverified", a.nick == "admin", created).Scan(&id)
		if err != nil {
			return err
		}
		s.users[a.nick] = id
		if a.topic == "" {
			continue
		}
		description := a.topic + "."
		if a.nick == alina {
			description += "\n\n" + sizedText(2300, studioParagraphs)
		}
		if _, err := s.tx.Exec(s.ctx, `
			INSERT INTO creator_profiles (user_id,display_name,description,created_at,updated_at)
			VALUES ($1,$2,$3,$4,$5)
		`, id, a.name, description, created, s.now); err != nil {
			return err
		}
	}
	return nil
}

type tierFixture struct {
	author, name, description string
	price                     int64
}

func (s *seeder) createTiers() error {
	workshopNotes := sizedText(3200, studioParagraphs)
	for _, t := range []tierFixture{
		{alina, bronze, "Анонсы прогулок, небольшие упражнения и открытые заметки студии.", 0},
		{alina, silver, "Еженедельный разбор городской серии и задания для собственной прогулки.", 29900},
		{alina, gold, "Исходники, схемы света и разборы контактных листов.\n\n" + workshopNotes, 79900},
		{alina, platinum, "Полный архив и подробные журналы подготовки выпусков.\n\n" + workshopNotes, 199900},
		{lev, levEntry, "Эскизы, заметки о чернилах и подборка упражнений.", 15050},
		{lev, levPro, "Полные рабочие листы и рецепты пигментов.", 50000},
		{ira, "Пробный обжиг", "Образцы глазури. Цена ниже минимального платежа.", 50},
		{ira, "Керамическая лаборатория", "Планы будущих экспериментов с керамикой.", 79000},
		{mark, archived, "Первый сезон коротких прогулок со звуком.", 25000},
		{mark, retired, "Старый архив шумов и голосов с выездов.", 53000},
		{mark, markUsed, "Разборы дорожек и полные записи интервью.", 42000},
		{mark, markFree, "Планы занятий по монтажу и склейкам.", 37000},
	} {
		author := s.users[t.author]
		row, err := s.q.CreateTier(s.ctx, store.CreateTierParams{AuthorID: author, Name: t.name, Description: t.description, PriceKopecks: t.price})
		if err != nil {
			return err
		}
		if _, err := s.tx.Exec(s.ctx, "UPDATE subscription_tiers SET created_at=$2 WHERE id=$1", row.ID, s.now.AddDate(0, -15, 0)); err != nil {
			return err
		}
		s.tiers[t.name] = row
	}
	return nil
}

func (s *seeder) archiveTiers() error {
	for _, name := range []string{archived, retired} {
		t := s.tiers[name]
		if _, err := s.q.ArchiveTier(s.ctx, store.ArchiveTierParams{ID: t.ID, AuthorID: t.AuthorID}); err != nil {
			return err
		}
	}
	return nil
}

const markdownPost = `# Утро у старого моста

**Погода:** слабый дождь. *Материал:* плёнка ISO 400.

## Маршрут

1. Начните от трамвайной остановки.
2. Пройдите вдоль реки до пристани.

- Широкий кадр перед выходом к воде.
- Детали на теневой стороне домов.

> Каждая фотография разговаривает с соседней.

` + "```text\nISO 400 · 1/125 с · f/5.6\n```" + `

| Точка | Свет |
| --- | --- |
| Мост | Мягкий |
| Пристань | Контровой |

[Список авторов](/authors)

![Градиент из мастерской](/api/v1/authors/seed-alina-verbova/images/cover)

<div data-seed-html="unsafe">Сырой HTML</div>
[Проверка опасной схемы](javascript:alert%281%29)

Оригиналы дорожек и пробные кадры доступны в приложенных файлах.`

const (
	rangePost   = 0
	photoPost   = 1
	audioPost   = 2
	videoPost   = 3
	expiredPost = 8
)

var alinaTitles = []string{
	"7 маршрутов для утренней съёмки", "Город после дождя 🌧️", "Трамвайное кольцо: дневник звука", "Закат над мастерскими",
	"Как читать контактный лист", "Рассеянный свет в подъезде", "Семейный архив на одной полке", "Двое на пустой остановке",
	"Переулки за рынком", "Смена объектива в тумане", "Серебряный отпечаток", "Откуда в кадре блики",
	"Мокрая брусчатка", "Снять лестницу снизу", "Маршрут по красным вывескам", "Свет между домами",
	"Сюжет из трёх кадров", "Проба бумаги", "Вечерняя пересадка", "Дом с зелёными ставнями",
	"Настройки для длинной выдержки", "Сквер в начале лета", "Контуры крыши", "Прогулка без штатива",
	"Двери и таблички", "Кадр из окна автобуса", "Одна улица на рассвете", "Съёмка в синий час",
	"Летние киоски", "Тропинка вдоль канала", "Окна напротив", "Три варианта одной серии",
	"Прогулка со старой мыльницей", "Утро у депо", "Афиши в пустом павильоне", "Переход через ручей",
	"Мастерская во дворе", "Чёрно-белый снег", "Город между дождями", "Остановка на окраине",
	"Кафе до открытия", "Большие формы на маленьком кадре", "Летний ветер у воды", "Зелёная линия горизонта",
	"Долго ждать человека в кадре", "Мелочи на подоконнике", "Фонарь вдалеке", "Камера в рюкзаке",
	"Отражения в стеклянном переходе", "Как выбрать первый кадр", "Старинные номера домов", "Уличный свет и плёнка",
	"Последний круг по набережной", "Полевая тетрадь за лето",
}

func (s *seeder) addPost(author, title, teaser, body, tier string, daysAgo int) (int64, error) {
	var id int64
	at := s.now.AddDate(0, 0, -max(1, daysAgo))
	publishedAt := instant(time.Time{})
	if daysAgo > 0 {
		publishedAt = instant(at)
	}
	err := s.tx.QueryRow(s.ctx, `
		INSERT INTO posts(author_id,title,teaser,body_markdown,minimum_tier_id,published_at,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id
	`, s.users[author], title, teaser, body, nullableID(s.tiers[tier].ID),
		publishedAt, at.AddDate(0, 0, -1), at).Scan(&id)
	return id, err
}

func (s *seeder) createPosts() error {
	cycle := []string{"", silver, "", gold, bronze, platinum, silver, ""}
	s.posts = make([]int64, len(alinaTitles))
	for i, title := range slices.Backward(alinaTitles) {
		tier := cycle[i%len(cycle)]
		body := fmt.Sprintf("## %s\n\n%s", title, studioParagraphs[i%len(studioParagraphs)])
		teaser := fmt.Sprintf("%s. Делюсь находками с прогулки и идеей для следующей серии.", title)
		switch i {
		case rangePost:
			body = markdownPost
		case photoPost, audioPost:
			teaser = ""
		case expiredPost:
			tier = silver
		case 9:
			title = sizedText(200, []string{"Маршрут вдоль реки и заметки о том, как меняется свет между мостом и старыми мастерскими по дороге от трамвайного кольца к набережной"})
		case 10:
			teaser = sizedText(2000, []string{"В выпуске разберём, какие отпечатки попадут в весеннюю подборку. Покажу три решения для одного материала и объясню, как подготовиться к обсуждению серии."})
			body = sizedText(5000, studioParagraphs)
		case 11:
			body = sizedText(99850, studioParagraphs)
		}
		id, err := s.addPost(alina, title, teaser, body, tier, 1+2*i)
		if err != nil {
			return err
		}
		s.posts[i] = id
	}
	for i, p := range []struct{ author, title, tier, body string }{
		{alina, "Черновик маршрута вдоль товарной ветки", "", "Отметить места, откуда видно старую станцию."},
		{alina, "Черновик разбора: контровый свет", gold, "Подобрать три кадра и сделать пробные отпечатки."},
		{lev, "Четыре вида бумаги для туши", "", "Сравнил фактуру листов на одном и том же рисунке."},
		{lev, "Пигмент из железного купороса", levEntry, "Рецептура, время выдержки и контрольный лист."},
		{lev, "Рисунок на мокром листе", levPro, sizedText(5000, []string{"Лист смачиваю губкой по всей ширине, затем проверяю, как быстро впитывается тушь. Кисть веду от края к краю, ширину линии отмечаю на отдельном листе."})},
		{lev, "Как хранить пробники туши", "", "Записываю состав и дату рядом с каждым образцом."},
		{lev, "Линия с переменным нажимом", levEntry, "Упражнение с тремя перьями и калькой."},
		{lev, "Черновик: таблица пигментов", levPro, "Добавить светлые образцы после высыхания."},
		{nina, "Книги, которые удобно читать в дороге", "", "Три небольшие книги и заметки о том, где я их дочитала."},
		{nina, "Прогулка по отделу старых карт", "", "В городской библиотеке сейчас выставляют атласы XIX века."},
		{nina, "Как устроен открытый книжный клуб", "", "Читаем один рассказ в неделю и делимся короткими заметками."},
		{nina, "Черновик: письмо книжному клубу", "", "Предложить встречи по субботам."},
		{mark, "Как звучит пустой двор", "", "Выложил карту акустических точек и план прогулки."},
		{mark, "Первый сезон: записи у старой плотины", archived, "Разговоры, вода и короткая пауза перед грозой."},
		{mark, "Архив: ночной вокзал", retired, "Две записи с платформы, где меняется эхо."},
		{mark, "Склейки и фоновый шум", markUsed, "Сравнение исходника и двух вариантов монтажа."},
		{mark, "Тишина между репликами", markUsed, sizedText(5000, []string{"При монтаже разговора оставляю паузу такой длины, чтобы она сохраняла ритм голоса. Меряю уровень фонового шума до и после каждой склейки."})},
		{mark, "Черновик: шорохи в саду", "", "Дописать таймкоды и порядок дорожек."},
	} {
		daysAgo := 75 - 4*i
		if strings.HasPrefix(p.title, "Черновик") {
			daysAgo = 0
		}
		teaser := fmt.Sprintf("«%s» — заметка автора по материалам последнего выпуска.", p.title)
		if _, err := s.addPost(p.author, p.title, teaser, p.body, p.tier, daysAgo); err != nil {
			return err
		}
	}
	for i, a := range catalog {
		if _, err := s.addPost(a.nick, "Первая заметка: "+a.topic, "Знакомство с проектом и планы ближайших выпусков.", "Здесь собираю идеи и короткие заметки. В первом выпуске рассказываю, с чего начались мои поиски по теме «"+a.topic+"».", "", 12+3*i); err != nil {
			return err
		}
		if _, err := s.addPost(a.nick, "Черновик следующего выпуска", "Продолжаю первый выпуск и собираю примеры.", "Проверить источники перед публикацией.", "", 0); err != nil {
			return err
		}
	}
	return nil
}
