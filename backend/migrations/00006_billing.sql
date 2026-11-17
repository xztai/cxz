-- +goose Up
-- Billing intent is separate from the paid access grant in subscriptions.
CREATE TABLE billing_agreements (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subscriber_id bigint NOT NULL REFERENCES users(id),
    author_id bigint NOT NULL REFERENCES creator_profiles(user_id),
    tier_id bigint NOT NULL,
    price_kopecks bigint NOT NULL CHECK (price_kopecks BETWEEN 100 AND 100000000),
    next_tier_id bigint,
    next_price_kopecks bigint CHECK (next_price_kopecks BETWEEN 100 AND 100000000),
    auto_renew boolean NOT NULL DEFAULT false,
    consent_at timestamptz,
    payment_method_id text NOT NULL DEFAULT '',
    paid_until timestamptz,
    anchor_day integer NOT NULL DEFAULT 1 CHECK (anchor_day BETWEEN 1 AND 31),
    stopped_reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (subscriber_id, author_id),
    CHECK (subscriber_id <> author_id),
    CHECK ((next_tier_id IS NULL) = (next_price_kopecks IS NULL)),
    FOREIGN KEY (tier_id, author_id) REFERENCES subscription_tiers(id, author_id),
    FOREIGN KEY (next_tier_id, author_id) REFERENCES subscription_tiers(id, author_id)
);
CREATE TABLE payments (
    id text PRIMARY KEY,
    agreement_id bigint NOT NULL REFERENCES billing_agreements(id),
    tier_id bigint NOT NULL REFERENCES subscription_tiers(id),
    amount_kopecks bigint NOT NULL CHECK (amount_kopecks BETWEEN 100 AND 100000000),
    renewal boolean NOT NULL,
    save_method boolean NOT NULL,
    request_body text NOT NULL,
    provider_id text NOT NULL DEFAULT '',
    confirmation_url text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','pending','succeeded','canceled','review')),
    created_at timestamptz NOT NULL DEFAULT now(),
    first_sent_at timestamptz,
    available_at timestamptz NOT NULL DEFAULT now(),
    lease_until timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    period_start timestamptz,
    period_end timestamptz,
    failure_code text NOT NULL DEFAULT '',
    attempts integer NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX payments_provider_unique ON payments(provider_id) WHERE provider_id <> '';
CREATE UNIQUE INDEX payments_one_open ON payments(agreement_id) WHERE status IN ('queued','pending','review');
CREATE INDEX payments_queue ON payments(available_at) WHERE status IN ('queued','pending');
CREATE INDEX payments_history ON payments(agreement_id,created_at DESC);
CREATE INDEX billing_due ON billing_agreements(paid_until) WHERE auto_renew;

-- Archive and block always win over future renewal scheduling, but preserve grants.
-- +goose StatementBegin
CREATE FUNCTION stop_billing_for_tier() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.archived_at IS NULL AND NEW.archived_at IS NOT NULL THEN
        UPDATE billing_agreements SET auto_renew=false, stopped_reason='tier_archived',
            next_tier_id=NULL,next_price_kopecks=NULL
        WHERE tier_id=NEW.id OR next_tier_id=NEW.id;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER stop_billing_for_tier AFTER UPDATE OF archived_at ON subscription_tiers
FOR EACH ROW EXECUTE FUNCTION stop_billing_for_tier();
-- +goose StatementBegin
CREATE FUNCTION stop_billing_for_user() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT OLD.blocked AND NEW.blocked THEN
        UPDATE billing_agreements SET auto_renew=false,stopped_reason='account_blocked',
            next_tier_id=NULL,next_price_kopecks=NULL
        WHERE subscriber_id=NEW.id OR author_id=NEW.id;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER stop_billing_for_user AFTER UPDATE OF blocked ON users
FOR EACH ROW EXECUTE FUNCTION stop_billing_for_user();
ALTER TABLE mail_outbox ADD COLUMN kind text NOT NULL DEFAULT 'action' CHECK(kind IN ('action','billing')),
 ADD COLUMN recipient_user_id bigint REFERENCES users(id);

-- +goose Down
ALTER TABLE mail_outbox DROP COLUMN kind, DROP COLUMN recipient_user_id;
DROP TRIGGER stop_billing_for_user ON users;
DROP FUNCTION stop_billing_for_user();
DROP TRIGGER stop_billing_for_tier ON subscription_tiers;
DROP FUNCTION stop_billing_for_tier();
DROP TABLE payments,billing_agreements;
