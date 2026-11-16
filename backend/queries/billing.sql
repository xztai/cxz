-- name: BillingTier :one
SELECT t.* FROM subscription_tiers t JOIN users u ON u.id=t.author_id
WHERE t.id=$1 AND t.archived_at IS NULL AND NOT u.blocked AND u.email_verified;

-- name: Agreement :one
SELECT * FROM billing_agreements WHERE subscriber_id=$1 AND author_id=$2;

-- name: LockAgreement :one
SELECT * FROM billing_agreements WHERE id=$1 FOR UPDATE;

-- name: CreateAgreement :one
INSERT INTO billing_agreements(subscriber_id,author_id,tier_id,price_kopecks)
VALUES($1,$2,$3,$4) ON CONFLICT(subscriber_id,author_id) DO UPDATE SET subscriber_id=EXCLUDED.subscriber_id RETURNING *;

-- name: PrepareAgreement :exec
UPDATE billing_agreements SET tier_id=$2,price_kopecks=$3,auto_renew=$4,
 consent_at=CASE WHEN $4 THEN now() ELSE NULL END,payment_method_id='',
 next_tier_id=NULL,next_price_kopecks=NULL,stopped_reason='' WHERE id=$1;

-- name: StopAgreement :exec
UPDATE billing_agreements SET auto_renew=false,stopped_reason=$2,next_tier_id=NULL,next_price_kopecks=NULL WHERE id=$1;

-- name: ChangeAgreementTier :exec
UPDATE billing_agreements SET next_tier_id=$2,next_price_kopecks=$3 WHERE id=$1;

-- name: CompleteAgreement :exec
UPDATE billing_agreements SET tier_id=$2,price_kopecks=$3,paid_until=$4,anchor_day=$5,
 payment_method_id=CASE WHEN sqlc.arg(method_id)::text <> '' THEN sqlc.arg(method_id)::text ELSE payment_method_id END,
 auto_renew=auto_renew AND (sqlc.arg(method_id)::text <> '' OR payment_method_id <> ''),
 next_tier_id=NULL,next_price_kopecks=NULL,stopped_reason='' WHERE id=$1;

-- name: PutPaidAccess :exec
INSERT INTO subscriptions(subscriber_id,author_id,tier_id,starts_at,expires_at)
VALUES($1,$2,$3,$4,$5) ON CONFLICT(subscriber_id,author_id) DO UPDATE
SET tier_id=EXCLUDED.tier_id,starts_at=EXCLUDED.starts_at,expires_at=EXCLUDED.expires_at,revoked_at=NULL;

-- name: InsertPayment :one
INSERT INTO payments(id,agreement_id,tier_id,amount_kopecks,renewal,save_method,request_body)
VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING *;

-- name: OpenPayment :one
SELECT * FROM payments WHERE agreement_id=$1 AND status IN ('queued','pending','review');

-- name: PaymentByID :one
SELECT * FROM payments WHERE id=$1;

-- name: PaymentByProvider :one
SELECT * FROM payments WHERE provider_id=$1;

-- name: PaymentOwner :one
SELECT a.* FROM payments p JOIN billing_agreements a ON a.id=p.agreement_id WHERE p.id=$1;

-- name: ClaimPayment :one
UPDATE payments SET lease_until=now()+interval '30 seconds',attempts=attempts+1
WHERE id=(SELECT id FROM payments WHERE status IN ('queued','pending') AND available_at<=now() AND lease_until<=now()
ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING *;

-- name: RetryPayment :exec
UPDATE payments SET available_at=now()+interval '30 seconds',lease_until=now(),failure_code=$2 WHERE id=$1 AND status IN ('queued','pending');

-- name: ReviewPayment :exec
UPDATE payments SET status='review',failure_code=$2,lease_until=now() WHERE id=$1 AND status IN ('queued','pending');

-- name: UpdateProviderPayment :exec
UPDATE payments SET provider_id=$2,confirmation_url=$3,status='pending',available_at=now()+interval '30 seconds',lease_until=now(),failure_code='' WHERE id=$1 AND status IN ('queued','pending','review');

-- name: FinishPayment :exec
UPDATE payments SET provider_id=$2,status=$3,completed_at=now(),period_start=$4,period_end=$5,failure_code=$6,lease_until=now()
WHERE id=$1;

-- name: LockPayment :one
SELECT * FROM payments WHERE id=$1 FOR UPDATE;

-- name: DueAgreementIDs :many
SELECT a.id FROM billing_agreements a JOIN users u ON u.id=a.subscriber_id JOIN users author ON author.id=a.author_id
JOIN subscription_tiers t ON t.id=a.tier_id
WHERE a.auto_renew AND a.payment_method_id<>'' AND a.paid_until<=now() AND NOT u.blocked AND NOT author.blocked
AND t.archived_at IS NULL AND NOT EXISTS(SELECT 1 FROM payments p WHERE p.agreement_id=a.id AND p.status IN ('queued','pending','review'))
ORDER BY a.paid_until LIMIT 100;

-- name: ListAgreements :many
SELECT a.id,a.author_id,u.nickname,p.display_name,a.tier_id,t.name AS tier_name,a.price_kopecks,
 a.auto_renew,a.paid_until,a.next_tier_id,a.next_price_kopecks,nt.name AS next_tier_name,a.stopped_reason,
 EXISTS(SELECT 1 FROM subscriptions s WHERE s.subscriber_id=a.subscriber_id AND s.author_id=a.author_id
 AND s.revoked_at IS NULL AND s.starts_at<=now() AND s.expires_at>now() AND NOT u.blocked) AS active
FROM billing_agreements a JOIN users u ON u.id=a.author_id JOIN creator_profiles p ON p.user_id=a.author_id
JOIN subscription_tiers t ON t.id=a.tier_id LEFT JOIN subscription_tiers nt ON nt.id=a.next_tier_id
WHERE a.subscriber_id=$1 ORDER BY a.id DESC LIMIT 25 OFFSET $2;

-- name: ListPayments :many
SELECT p.id,p.agreement_id,p.amount_kopecks,p.status,p.renewal,p.created_at,p.period_start,p.period_end,p.failure_code,
 u.nickname,cp.display_name,t.name AS tier_name
FROM payments p JOIN billing_agreements a ON a.id=p.agreement_id JOIN users u ON u.id=a.author_id
JOIN creator_profiles cp ON cp.user_id=a.author_id JOIN subscription_tiers t ON t.id=p.tier_id
WHERE a.subscriber_id=$1 ORDER BY p.created_at DESC,p.id DESC LIMIT 25 OFFSET $2;

-- name: QueueBillingMail :exec
INSERT INTO mail_outbox(recipient,subject,body,token_hash,expires_at,kind,recipient_user_id)
VALUES($1,$2,$3,'',now()+interval '7 days','billing',$4);

-- name: BillingMailActive :one
SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND NOT blocked AND email_verified);

-- name: AgreementAuthor :one
SELECT author_id FROM billing_agreements WHERE id=$1;

-- name: BillingUserActive :one
SELECT NOT blocked AND email_verified FROM users WHERE id=$1;

-- name: BillingRecipient :one
SELECT id,email FROM users WHERE id=$1;

-- name: StartPaymentSend :one
UPDATE payments SET first_sent_at=COALESCE(first_sent_at,now()) WHERE id=$1 RETURNING *;
