CREATE TABLE IF NOT EXISTS cms.homepage_layout (
    id BIGSERIAL PRIMARY KEY,
    draft_sections JSONB NOT NULL DEFAULT '[]'::jsonb,
    published_sections JSONB NOT NULL DEFAULT '[]'::jsonb,
    published_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by TEXT
);

INSERT INTO cms.homepage_layout (id) VALUES (1) ON CONFLICT DO NOTHING;
