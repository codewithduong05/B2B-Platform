-- TASK-020: M5 CMS Slice 2 — SEO Templates, Sitemap, Structured Data, Product Feed

CREATE TABLE IF NOT EXISTS cms.seo_template (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    title_template TEXT NOT NULL,
    description_template TEXT,
    structured_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_seo_template_key ON cms.seo_template (key);

CREATE TABLE IF NOT EXISTS cms.seo_settings (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_seo_settings_key ON cms.seo_settings (key);
