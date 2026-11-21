package main

import (
	"errors"
	"fmt"
	"strings"

	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
)

func (s *seeder) reset() error {
	_, err := s.tx.Exec(s.ctx, `
		LOCK TABLE users, creator_profiles, posts, profile_images, post_media, post_likes, post_comments,
		   billing_agreements, payments, subscriptions, subscription_tiers IN SHARE ROW EXCLUSIVE MODE
	`)
	if err != nil {
		return phaseError{"блокировка таблиц для reset", err}
	}
	var ids []int64
	if err := s.tx.QueryRow(s.ctx, `
		SELECT ARRAY(SELECT id FROM users WHERE right(email,length($1))=$1)
	`, emailSuffix).Scan(&ids); err != nil {
		return phaseError{"поиск аккаунтов для reset", err}
	}
	if len(ids) == 0 {
		return nil
	}
	var crossAccount bool
	err = s.tx.QueryRow(s.ctx, `
		SELECT EXISTS(
			SELECT 1 FROM billing_agreements WHERE (subscriber_id=ANY($1))<>(author_id=ANY($1))
			UNION ALL SELECT 1 FROM subscriptions WHERE (subscriber_id=ANY($1))<>(author_id=ANY($1))
			UNION ALL SELECT 1 FROM post_likes l JOIN posts p ON p.id=l.post_id
				WHERE p.author_id=ANY($1) AND l.user_id<>ALL($1)
			UNION ALL SELECT 1 FROM post_comments c JOIN posts p ON p.id=c.post_id
				WHERE p.author_id=ANY($1) AND c.user_id<>ALL($1)
		)
	`, ids).Scan(&crossAccount)
	if err != nil {
		return phaseError{"проверка связей с обычными аккаунтами", err}
	}
	if crossAccount {
		return errors.New("reset остановлен: найдены подписки, соглашения или реакции обычных аккаунтов, связанные с сидом")
	}
	for _, statement := range []string{
		"DELETE FROM payments WHERE agreement_id IN (SELECT id FROM billing_agreements WHERE subscriber_id=ANY($1) AND author_id=ANY($1))",
		"DELETE FROM billing_agreements WHERE subscriber_id=ANY($1) AND author_id=ANY($1)",
		"DELETE FROM subscriptions WHERE subscriber_id=ANY($1) AND author_id=ANY($1)",
		"DELETE FROM post_likes WHERE user_id=ANY($1)",
		"DELETE FROM post_comments WHERE user_id=ANY($1)",
		"DELETE FROM post_media WHERE author_id=ANY($1)",
		"DELETE FROM posts WHERE author_id=ANY($1)",
		"DELETE FROM mail_outbox WHERE recipient_user_id=ANY($1)",
	} {
		if _, err := s.tx.Exec(s.ctx, statement, ids); err != nil {
			return phaseError{"очистка связей сидовых аккаунтов", err}
		}
	}
	for _, statement := range []string{
		"DELETE FROM mail_outbox WHERE right(recipient,length($1))=$1",
		"DELETE FROM rate_limits WHERE right(key,length($1))=$1",
	} {
		if _, err := s.tx.Exec(s.ctx, statement, emailSuffix); err != nil {
			return phaseError{"очистка очереди и лимитов сидовых аккаунтов", err}
		}
	}

	// Deleting a used tier is forbidden by design. Suspend exactly that guard
	// for this transaction's domain-scoped delete. All foreign keys remain active.
	// PostgreSQL rolls back both the delete and the trigger state on failure.
	var enabled bool
	if err := s.tx.QueryRow(s.ctx, `
		SELECT tgenabled='O' FROM pg_trigger WHERE tgrelid='subscription_tiers'::regclass
		    AND tgname='protect_used_tier'
	`).Scan(&enabled); err != nil || !enabled {
		return phaseError{"проверка защитного триггера уровней", err}
	}
	if _, err := s.tx.Exec(s.ctx, "ALTER TABLE subscription_tiers DISABLE TRIGGER protect_used_tier"); err != nil {
		return phaseError{"reset требует роль владельца таблиц", err}
	}
	if _, err := s.tx.Exec(s.ctx, "DELETE FROM subscription_tiers WHERE author_id=ANY($1)", ids); err != nil {
		return phaseError{"очистка уровней сидовых авторов", err}
	}
	if _, err := s.tx.Exec(s.ctx, "ALTER TABLE subscription_tiers ENABLE TRIGGER protect_used_tier"); err != nil {
		return phaseError{"восстановление защитного триггера", err}
	}
	if _, err := s.tx.Exec(s.ctx, "DELETE FROM users WHERE id=ANY($1)", ids); err != nil {
		return phaseError{"очистка сидовых аккаунтов", err}
	}
	return nil
}

const owned = "WITH owned AS (SELECT id FROM users WHERE right(email,length($1))=$1) "

func (s *seeder) summary() error {
	rows, err := s.tx.Query(s.ctx, owned+`
		SELECT 'users',count(*) FROM owned
		UNION ALL SELECT 'creator_profiles',count(*) FROM creator_profiles WHERE user_id IN (SELECT id FROM owned)
		UNION ALL SELECT 'profile_images',count(*) FROM profile_images WHERE user_id IN (SELECT id FROM owned)
		UNION ALL SELECT 'subscription_tiers',count(*) FROM subscription_tiers WHERE author_id IN (SELECT id FROM owned)
		UNION ALL SELECT 'posts (опубликованные)',count(*) FROM posts WHERE author_id IN (SELECT id FROM owned) AND published_at IS NOT NULL
		UNION ALL SELECT 'posts (черновики)',count(*) FROM posts WHERE author_id IN (SELECT id FROM owned) AND published_at IS NULL
		UNION ALL SELECT 'post_media',count(*) FROM post_media WHERE author_id IN (SELECT id FROM owned)
		UNION ALL SELECT 'billing_agreements',count(*) FROM billing_agreements WHERE subscriber_id IN (SELECT id FROM owned)
		UNION ALL SELECT 'subscriptions',count(*) FROM subscriptions WHERE subscriber_id IN (SELECT id FROM owned)
		UNION ALL SELECT 'payments',count(*) FROM payments WHERE agreement_id IN
		    (SELECT id FROM billing_agreements WHERE subscriber_id IN (SELECT id FROM owned))
		UNION ALL SELECT 'payments (' || status || ')',count(*) FROM payments WHERE agreement_id IN
		    (SELECT id FROM billing_agreements WHERE subscriber_id IN (SELECT id FROM owned)) GROUP BY status
		UNION ALL SELECT 'post_likes',count(*) FROM post_likes WHERE user_id IN (SELECT id FROM owned)
		UNION ALL SELECT 'post_comments',count(*) FROM post_comments WHERE user_id IN (SELECT id FROM owned)
	`, emailSuffix)
	if err != nil {
		return phaseError{"сводка сидовых строк", err}
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var count int64
		if err := rows.Scan(&name, &count); err != nil {
			return phaseError{"чтение сводки", err}
		}
		fmt.Printf("  %-27s %d\n", name, count)
	}
	if err := rows.Err(); err != nil {
		return phaseError{"чтение сводки", err}
	}
	return nil
}

type invariant struct {
	label, query string
}

var invariants = []invariant{
	{"использование уровней отмечено триггерами, позиции уникальны", `
		, refs AS (
		  SELECT minimum_tier_id AS id FROM posts WHERE author_id IN (SELECT id FROM owned)
		  UNION SELECT tier_id FROM subscriptions WHERE author_id IN (SELECT id FROM owned)
		  UNION SELECT p.tier_id FROM payments p JOIN billing_agreements a ON a.id=p.agreement_id
		    WHERE a.author_id IN (SELECT id FROM owned)
		  UNION SELECT next_tier_id FROM billing_agreements WHERE author_id IN (SELECT id FROM owned)
		)
		SELECT NOT EXISTS(SELECT 1 FROM subscription_tiers t LEFT JOIN refs r ON r.id=t.id
		    WHERE t.author_id IN (SELECT id FROM owned) AND (t.used_at IS NOT NULL)<>(r.id IS NOT NULL))
		AND NOT EXISTS(SELECT 1 FROM subscription_tiers WHERE author_id IN (SELECT id FROM owned)
		    GROUP BY author_id,position HAVING count(*)>1)
		AND NOT EXISTS(SELECT 1 FROM subscription_tiers WHERE author_id IN (SELECT id FROM owned)
		    AND archived_at IS NULL GROUP BY author_id HAVING count(*)>50)
	`},
	{"подписки и сроки совпадают с оплаченными периодами", `
		SELECT NOT EXISTS(
		    SELECT 1 FROM subscriptions s JOIN billing_agreements a USING (subscriber_id,author_id)
		    WHERE s.subscriber_id IN (SELECT id FROM owned)
		    AND (s.expires_at IS DISTINCT FROM a.paid_until OR s.tier_id<>a.tier_id OR NOT EXISTS(
		        SELECT 1 FROM payments p WHERE p.agreement_id=a.id AND p.status='succeeded'
		            AND p.tier_id=s.tier_id AND p.period_start=s.starts_at AND p.period_end=s.expires_at)))
		AND NOT EXISTS(SELECT 1 FROM subscriptions s WHERE s.subscriber_id IN (SELECT id FROM owned)
		    AND NOT EXISTS(SELECT 1 FROM billing_agreements a WHERE a.subscriber_id=s.subscriber_id AND a.author_id=s.author_id))
		AND NOT EXISTS(SELECT 1 FROM billing_agreements a WHERE a.subscriber_id IN (SELECT id FROM owned)
		    AND a.paid_until IS NOT NULL AND NOT EXISTS(
		        SELECT 1 FROM subscriptions s WHERE s.subscriber_id=a.subscriber_id AND s.author_id=a.author_id))
	`},
	{"платежи в RUB, тестовые, с зафиксированной суммой и без пересечения периодов", `
		, attempts AS (
		    SELECT p.*, a.author_id, t.price_kopecks, t.author_id AS tier_author
		    FROM payments p JOIN billing_agreements a ON a.id=p.agreement_id JOIN subscription_tiers t ON t.id=p.tier_id
		    WHERE a.subscriber_id IN (SELECT id FROM owned)
		)
		SELECT NOT EXISTS(SELECT 1 FROM attempts p WHERE amount_kopecks<>price_kopecks OR author_id<>tier_author
		    OR (request_body::jsonb->>'test') IS DISTINCT FROM 'true'
		    OR (request_body::jsonb#>>'{amount,currency}') IS DISTINCT FROM 'RUB'
		    OR ((request_body::jsonb#>>'{amount,value}')::numeric*100) IS DISTINCT FROM amount_kopecks::numeric
		    OR (status='succeeded' AND (period_start IS NULL OR period_end IS NULL OR period_end<=period_start
		        OR completed_at IS NULL OR completed_at>now() OR provider_id=''))
		    OR (status<>'succeeded' AND (period_start IS NOT NULL OR period_end IS NOT NULL))
		    OR (status IN ('queued','pending') AND available_at<>'infinity'))
		AND NOT EXISTS(SELECT 1 FROM attempts p JOIN attempts q ON p.agreement_id=q.agreement_id AND p.id<q.id
		    WHERE p.period_start<q.period_end AND q.period_start<p.period_end)
		AND (SELECT count(DISTINCT status)=5 FROM attempts)
		AND (SELECT count(DISTINCT failure_code)>=2 FROM attempts WHERE status='canceled')
	`},
	{"у лайков есть доступ, комментарии опубликованы при действовавшем доступе", `
		SELECT NOT EXISTS(SELECT 1 FROM post_likes l JOIN posts p ON p.id=l.post_id
		    WHERE p.author_id IN (SELECT id FROM owned)
		    AND (NOT post_readable(p,l.user_id) OR l.created_at<=p.published_at OR l.created_at>now()))
		AND NOT EXISTS(SELECT 1 FROM post_comments c JOIN posts p ON p.id=c.post_id
		    WHERE p.author_id IN (SELECT id FROM owned) AND (
		     c.created_at<=p.published_at OR c.created_at>now()
		     OR (p.minimum_tier_id IS NOT NULL AND p.author_id<>c.user_id AND NOT EXISTS(
		        SELECT 1 FROM billing_agreements a JOIN payments paid ON paid.agreement_id=a.id
		        JOIN subscription_tiers grant_tier ON grant_tier.id=paid.tier_id
		        JOIN subscription_tiers need ON need.id=p.minimum_tier_id
		        WHERE a.subscriber_id=c.user_id AND a.author_id=p.author_id AND paid.status='succeeded'
		        AND grant_tier.position>=need.position AND paid.period_start<=c.created_at AND paid.period_end>c.created_at))))
		AND (SELECT count(*)>=3 FROM post_comments c JOIN posts p ON p.id=c.post_id
		    WHERE c.user_id=(SELECT id FROM users WHERE nickname='seed-pavel-gromov')
		    AND NOT post_readable(p,c.user_id))
	`},
	{"архивные уровни сохраняют доступ, соседние активные уровни различаются по used", `
		SELECT (SELECT count(*)=2 FROM subscription_tiers t JOIN users u ON u.id=t.author_id
		    WHERE u.nickname='seed-mark-volkov' AND t.archived_at IS NOT NULL AND t.used_at IS NOT NULL)
		AND EXISTS(SELECT 1 FROM subscription_tiers t JOIN users u ON u.id=t.author_id
		    JOIN subscription_tiers free ON free.author_id=t.author_id AND free.position=t.position+1
		    WHERE u.nickname='seed-mark-volkov' AND t.archived_at IS NULL AND free.archived_at IS NULL
		    AND t.used_at IS NOT NULL AND free.used_at IS NULL AND t.price_kopecks>free.price_kopecks)
		AND EXISTS(SELECT 1 FROM subscriptions s JOIN subscription_tiers t ON t.id=s.tier_id
		    JOIN billing_agreements a ON a.subscriber_id=s.subscriber_id AND a.author_id=s.author_id
		    WHERE s.subscriber_id IN (SELECT id FROM owned) AND t.archived_at IS NOT NULL
		    AND s.starts_at<=now() AND s.expires_at>now() AND NOT a.auto_renew AND a.stopped_reason='tier_archived')
	`},
	{"аналитика покрывает 90 дней и разные суммы, история читателя содержит 60+ платежей", `
		SELECT EXISTS(SELECT 1 FROM payments p JOIN billing_agreements a ON a.id=p.agreement_id
		    JOIN users author ON author.id=a.author_id
		    WHERE author.nickname='seed-alina-verbova' AND p.status='succeeded' AND p.completed_at>now()-interval '90 days'
		    HAVING count(DISTINCT (p.completed_at AT TIME ZONE 'UTC')::date) BETWEEN 30 AND 70
		       AND count(DISTINCT p.amount_kopecks)>=3 AND min(p.completed_at)<=now()-interval '85 days')
		AND EXISTS(SELECT 1 FROM payments p JOIN billing_agreements a ON a.id=p.agreement_id
		    JOIN users author ON author.id=a.author_id WHERE author.nickname='seed-lev-stone'
		    AND p.status='succeeded' AND p.completed_at>now()-interval '30 days' HAVING count(*) BETWEEN 18 AND 23)
		AND EXISTS(SELECT 1 FROM payments p JOIN billing_agreements a ON a.id=p.agreement_id
		    JOIN users u ON u.id=a.subscriber_id WHERE u.id IN (SELECT id FROM owned)
		    AND u.nickname='seed-denis-orlov' HAVING count(*)>=60)
	`},
}

func (s *seeder) verify() error {
	for _, check := range invariants {
		var ok bool
		if err := s.tx.QueryRow(s.ctx, owned+strings.TrimSpace(check.query), emailSuffix).Scan(&ok); err != nil {
			return phaseError{"проверка: " + check.label, err}
		}
		if !ok {
			return errors.New("нарушена проверка: " + check.label)
		}
		fmt.Println("✓", check.label)
	}
	for _, name := range []string{readers[0].nick, readers[1].nick, readers[4].nick, "reader"} {
		rows, err := s.q.ListFeed(s.ctx, store.ListFeedParams{ViewerID: s.users[name]})
		if err != nil {
			return phaseError{"запрос ленты " + name, err}
		}
		page := rows[:min(24, len(rows))]
		readable := 0
		for _, row := range page {
			if row.CanRead {
				readable++
			}
		}
		if (name == "reader" && len(page) != 0) || (name != "reader" && (readable == 0 || readable == len(page))) {
			return errors.New("неожиданный доступ в ленте " + name)
		}
		fmt.Printf("✓ лента seed-%s: %d доступных, %d закрытых\n", name, readable, len(page)-readable)
	}
	for key := range s.keys {
		file, err := s.s3.Open(s.ctx, key)
		if err != nil {
			return phaseError{"проверка загруженного объекта в S3", err}
		}
		if err := file.Close(); err != nil {
			return phaseError{"закрытие объекта S3", err}
		}
	}
	fmt.Printf("✓ S3: %d объектов, включая миниатюры\n", len(s.keys))
	return nil
}

func (s *seeder) pruneMedia(conn *pgx.Conn) error {
	q := store.New(conn)
	for _, prefix := range []string{postPrefix, profilePrefix} {
		after := ""
		for {
			page, err := s.s3.ListPage(s.ctx, prefix, after, 64)
			if err != nil {
				return phaseError{"поиск устаревших объектов сида", err}
			}
			if len(page) == 0 {
				break
			}
			var candidates []string
			for _, obj := range page {
				if !s.keys[obj.Key] {
					candidates = append(candidates, obj.Key)
				}
			}
			if len(candidates) > 0 {
				refs, err := q.ReferencedMediaKeys(s.ctx, candidates)
				if err != nil {
					return phaseError{"проверка ссылок на устаревшие объекты", err}
				}
				refSet := make(map[string]bool, len(refs))
				for _, key := range refs {
					refSet[key] = true
				}
				for _, key := range candidates {
					if !refSet[key] {
						if err := s.s3.Delete(s.ctx, key); err != nil {
							return phaseError{"удаление устаревшего объекта сида", err}
						}
					}
				}
			}
			after = page[len(page)-1].Key
		}
	}
	return nil
}
