-- Rollback M5 CMS Slice 9: FAQ category column

DROP INDEX IF EXISTS idx_cms_faq_category;
ALTER TABLE cms.faq DROP COLUMN IF EXISTS category;