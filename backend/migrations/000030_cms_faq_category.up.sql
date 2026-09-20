-- M5 CMS Slice 9: FAQ category column

ALTER TABLE cms.faq ADD COLUMN IF NOT EXISTS category VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_cms_faq_category ON cms.faq (category) WHERE deleted_at IS NULL;