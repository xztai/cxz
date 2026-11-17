-- name: PaymentDailyStats :many
WITH confirmed AS (
 SELECT (p.completed_at AT TIME ZONE 'UTC')::date AS day,
  count(*)::bigint AS payment_count,sum(p.amount_kopecks)::bigint AS amount_kopecks
 FROM payments p JOIN billing_agreements a ON a.id=p.agreement_id
 WHERE a.author_id=sqlc.arg(author_id) AND p.status='succeeded' AND p.provider_id<>''
 AND p.completed_at>=sqlc.arg(period_from)::timestamptz AND p.completed_at<sqlc.arg(period_to)::timestamptz
 GROUP BY (p.completed_at AT TIME ZONE 'UTC')::date
), days AS (
 SELECT generate_series(0,sqlc.arg(day_count)::integer-1)::integer AS n
)
SELECT to_char((sqlc.arg(period_from)::timestamptz AT TIME ZONE 'UTC')::date+days.n,'YYYY-MM-DD')::text AS day,
 COALESCE(c.payment_count,0)::bigint AS payment_count,COALESCE(c.amount_kopecks,0)::bigint AS amount_kopecks
FROM days LEFT JOIN confirmed c ON c.day=(sqlc.arg(period_from)::timestamptz AT TIME ZONE 'UTC')::date+days.n
ORDER BY days.n;

-- name: ActiveSubscriptionsByTier :many
SELECT t.id,t.name,(t.archived_at IS NOT NULL)::boolean AS archived,count(sub.subscriber_id)::bigint AS subscriber_count
FROM subscription_tiers t
LEFT JOIN subscriptions sub ON sub.tier_id=t.id AND sub.author_id=t.author_id AND sub.revoked_at IS NULL
 AND sub.starts_at<=now() AND sub.expires_at>now()
 AND EXISTS(SELECT 1 FROM users u WHERE u.id=sub.subscriber_id AND NOT u.blocked AND u.email_verified)
WHERE t.author_id=$1
GROUP BY t.id ORDER BY t.position DESC;
