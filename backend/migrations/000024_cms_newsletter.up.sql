CREATE TABLE IF NOT EXISTS cms.newsletter_subscriber (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'unsubscribed')),
    first_name TEXT,
    source TEXT,
    confirmation_token_hash TEXT,
    token_expires_at TIMESTAMPTZ,
    unsubscribe_token_hash TEXT,
    unsubscribe_token_expires_at TIMESTAMPTZ,
    subscribed_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    unsubscribed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_newsletter_subscriber_status ON cms.newsletter_subscriber (status);
CREATE INDEX IF NOT EXISTS idx_newsletter_subscriber_created_at ON cms.newsletter_subscriber (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_newsletter_subscriber_email ON cms.newsletter_subscriber (email);
