package main

import (
	"slices"
	"time"
)

var comments = []string{
	"Классно, что в примерах есть кадры без обработки. Сразу понятнее, откуда берётся этот цвет.",
	"Заметила на третьем кадре отражение в трамвайном окне. Это случайность или вы заранее ждали нужного момента?",
	"У меня от этого места всегда ощущение раннего воскресенья. Спасибо, что удалось его сохранить.",
	"Попробовал повторить маршрут на прошлых выходных. У воды действительно совсем другой свет.",
	"Хочу сверить настройки: в этот раз при замере вы смотрели на тени или на мокрый асфальт?",
	"Забрала упражнение себе на субботу. У меня как раз почти закончилась плёнка.",
	"Пробный отпечаток смотрится намного теплее. Интересно, как бы он выглядел на матовой бумаге.",
	"На маленьком экране я пропустил детали в витрине, а с ноутбука они сразу бросились в глаза.",
	"Мне понравился порядок кадров. После пустой остановки движение особенно хорошо чувствуется.",
	"Короткая заметка, но из тех, которые хочется перечитать перед прогулкой.",
	"В наших краях такой свет бывает ближе к вечеру. Проверю, получится ли похожий кадр.",
	"Благодаря этому разбору наконец понял, зачем оставлять в серии промежуточный кадр.",
	"Мне было бы интересно увидеть серию после дождя и в сухой день в одном выпуске.",
	"Вторая фотография нравится больше всего.",
	"Оказывается, я проходила мимо этой стены лет пять и ни разу не смотрела наверх.",
	"Упражнение с одним фокусным помогло убрать лишнее из кадра. Оставлю себе это ограничение на неделю.",
	"Поставила звук потише и всё равно хорошо слышно, как меняется фон под аркой.",
	"Спасибо за таймкоды. Удобно сначала послушать весь проход, а потом вернуться к деталям.",
	"На отпечатке набережная кажется шире за счёт пустого переднего плана. Хороший пример для нашей группы.",
	"Отдельное спасибо за маленькую схему: теперь понятно, куда уходят линии.",
	"На телефоне долго листала между снимками. На большом экране переход явно получается мягче.",
	"В следующий раз возьму вместо зума старый фикс. Как раз повод проверить дистанцию.",
	"У нас такую плитку после ремонта заменили. Приятно видеть, что где-то она ещё осталась.",
	"Сначала слушал в наушниках, потом через колонки. Шаги по металлу на колонках звучат иначе.",
	"Сделала пару тестовых кадров прямо после чтения. Тени в самом деле удерживают взгляд.",
	"Очень нравится, как вы объясняете выбор экспозиции. Спокойно и с конкретными примерами.",
	"Наконец пригодился блокнот с настройками. Сравнение через неделю работает лучше, чем я думал.",
	"На том же переходе мне мешают рекламные щиты. Попробую подняться к перилам повыше.",
	"Сохраню подборку для встречи фотоклуба, у нас как раз обсуждение серий в четверг.",
	"У воды на прошлой неделе было очень туманно. Любопытно сравнить с этим выпуском.",
	"Хороший повод выйти из дома пораньше.",
	"Вернулся к первому кадру после финального и заметил ещё одну перекличку по цвету.",
	"Здесь таксист на заднем плане почти незаметен, а теперь кажется главным героем.",
	"Поймала похожий блик на знакомой улице. Впервые осознанно оставила его в кадре.",
	"Спасибо за возможность скачать материалы. Рассмотрю увеличения на большом мониторе.",
}

func (s *seeder) createInteractions() error {
	for i, post := range s.posts {
		limit := i % 4
		switch i {
		case rangePost:
			limit = 37
		case photoPost:
			limit = 12
		case audioPost:
			limit = 25
		}
		_, err := s.tx.Exec(s.ctx, `
			INSERT INTO post_likes(post_id,user_id,created_at)
			SELECT p.id,u.id,clock.started+(now()-clock.started)*(0.25+0.1*(get_byte(decode(md5(u.nickname), 'hex'), 0)%6)::double precision)
			FROM posts p
			JOIN users u ON right(u.email,length($2))=$2 AND u.email_verified
			LEFT JOIN subscriptions sub ON sub.author_id=p.author_id AND sub.subscriber_id=u.id
			CROSS JOIN LATERAL (SELECT CASE WHEN p.minimum_tier_id IS NULL OR p.author_id=u.id
				THEN p.published_at ELSE greatest(p.published_at,sub.starts_at) END AS started) clock
			WHERE p.id=$1 AND post_readable(p,u.id)
			ORDER BY md5(u.nickname || p.title) LIMIT $3
		`, post, emailSuffix, limit)
		if err != nil {
			return err
		}
	}
	_, err := s.tx.Exec(s.ctx, `
		INSERT INTO post_likes(post_id,user_id,created_at)
		SELECT p.id,u.id,p.published_at+(now()-p.published_at)*0.4
		FROM posts p JOIN users author ON author.id=p.author_id
		JOIN users u ON u.nickname='seed-reader'
		WHERE right(author.email,length($1))=$1 AND author.nickname<>'seed-alina-verbova'
		 AND p.published_at IS NOT NULL AND p.minimum_tier_id IS NULL AND get_byte(decode(md5(p.title), 'hex'), 0)%3=0 AND post_readable(p,u.id)
	`, emailSuffix)
	if err != nil {
		return err
	}
	for _, i := range []int{rangePost, photoPost, expiredPost} {
		if err := s.commentThread(i); err != nil {
			return err
		}
	}
	return nil
}

type commenter struct {
	id    int64
	since time.Time
}

type commentFixture struct {
	user int64
	at   time.Time
	body string
}

func (s *seeder) commentThread(index int) error {
	postID := s.posts[index]
	var people []commenter
	rows, err := s.tx.Query(s.ctx, `
		SELECT u.id,CASE WHEN p.minimum_tier_id IS NULL OR p.author_id=u.id THEN p.published_at
		   ELSE greatest(p.published_at,sub.starts_at) END
		FROM posts p JOIN users u ON right(u.email,length($2))=$2 AND u.email_verified
		LEFT JOIN subscriptions sub ON sub.author_id=p.author_id AND sub.subscriber_id=u.id
		WHERE p.id=$1 AND post_readable(p,u.id)
		ORDER BY md5(u.nickname || p.title) LIMIT 22
	`, postID, emailSuffix)
	if err != nil {
		return err
	}
	for rows.Next() {
		var person commenter
		if err := rows.Scan(&person.id, &person.since); err != nil {
			rows.Close()
			return err
		}
		people = append(people, person)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	var thread []commentFixture
	for i := range len(comments) {
		p := people[i%len(people)]
		at := p.since.Add(s.now.Sub(p.since) * time.Duration(i+1) / time.Duration(len(comments)+1))
		thread = append(thread, commentFixture{p.id, at, comments[(i+index)%len(comments)]})
	}
	p := people[0]
	longComment := sizedText(2000, []string{
		"Делюсь, как это упражнение получилось у нас. Мы гуляли вдвоём и условились трижды вернуться на одну точку, записывая время и направление света. На второй круг оставили только один объектив.",
		"Когда разложили кадры рядом, стало видно, что устойчивее всего работает точка у остановки. Отражение в луже жило всего несколько минут. Остальные детали оставались почти одинаковыми, пока не вышло солнце.",
		"В следующий раз попробуем собрать из этих кадров короткую историю. Для меня пока самая трудная часть — оставить в серии кадр, который помогает связать соседние, хотя сам по себе выглядит скромно.",
	})
	thread = append(thread, commentFixture{p.id, p.since.Add(s.now.Sub(p.since) / 2), longComment})
	if index == expiredPost {
		for i, body := range []string{
			"Из всех вариантов выбрал съёмку из арки. На соседней улице есть похожий выход к рынку.",
			"На пробе с плёнкой сработало. Вылез интересный тёплый оттенок на мокрой стене.",
			"Пересмотрел серию на большом экране. Больше всего теперь нравится кадр с открытым окном.",
		} {
			at := s.now.AddDate(0, 0, -15+i*3)
			thread = append(thread, commentFixture{s.users[readers[3].nick], at, body})
		}
	}
	slices.SortFunc(thread, func(a, b commentFixture) int { return a.at.Compare(b.at) })
	for _, c := range thread {
		if _, err := s.tx.Exec(s.ctx, `
			INSERT INTO post_comments(post_id,user_id,body,created_at) VALUES($1,$2,$3,$4)
		`, postID, c.user, c.body, c.at); err != nil {
			return err
		}
	}
	return nil
}
