package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type CMSRepository struct {
	db *database.DB
	tx *database.Tx
}

func NewCMSRepository(db *database.DB) *CMSRepository {
	return &CMSRepository{db: db}
}

func NewCMSRepositoryWithTx(tx *database.Tx) *CMSRepository {
	return &CMSRepository{tx: tx}
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *CMSRepository) conn() querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db.Pool
}

type Article struct {
	ID          int64
	Code        string
	Slug        string
	Title       string
	Excerpt     *string
	Body        string
	Author      *string
	Status      string
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Page struct {
	ID              int64
	Code            string
	Slug            string
	Title           string
	Body            string
	MetaTitle       *string
	MetaDescription *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Faq struct {
	ID        int64
	Code      string
	Question  string
	Answer    string
	Category  *string
	SortOrder int
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Banner struct {
	ID        int64
	Code      string
	Title     string
	ImageUrl  string
	LinkUrl   *string
	Position  string
	SortOrder int
	IsActive  bool
	CreatedAt time.Time
}

type MenuItem struct {
	ID        int64
	Code      string
	Location  string
	Label     string
	Url       string
	ParentID  *int64
	SortOrder int
	IsActive  bool
	CreatedAt time.Time
}

type Setting struct {
	Key       string
	Value     string
	GroupName string
	UpdatedAt time.Time
}

const articleColumns = `id, code, slug, title, excerpt, body, author, status, published_at, created_at, updated_at`

func scanArticle(row interface{ Scan(...any) error }) (Article, error) {
	var a Article
	var excerpt, author pgtype.Text
	var pubAt pgtype.Timestamptz
	err := row.Scan(&a.ID, &a.Code, &a.Slug, &a.Title, &excerpt, &a.Body, &author, &a.Status, &pubAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, err
	}
	if excerpt.Valid {
		s := excerpt.String
		a.Excerpt = &s
	}
	if author.Valid {
		s := author.String
		a.Author = &s
	}
	if pubAt.Valid {
		t := pubAt.Time
		a.PublishedAt = &t
	}
	return a, nil
}

func (r *CMSRepository) CreateArticle(ctx context.Context, code, slug, title string, excerpt *string, body string, author *string) (Article, error) {
	var exc, auth pgtype.Text
	if excerpt != nil {
		exc = pgtype.Text{String: *excerpt, Valid: true}
	}
	if author != nil {
		auth = pgtype.Text{String: *author, Valid: true}
	}
	var id int64
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.article (code, slug, title, excerpt, body, author, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'draft')
		RETURNING id
	`, code, slug, title, exc, body, auth).Scan(&id)
	if err != nil {
		return Article{}, err
	}
	return r.GetArticleByID(ctx, id)
}

func (r *CMSRepository) GetArticleByID(ctx context.Context, id int64) (Article, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+articleColumns+` FROM cms.article WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanArticle(row)
}

func (r *CMSRepository) GetArticleBySlug(ctx context.Context, slug string, publishedOnly bool) (Article, error) {
	query := `SELECT ` + articleColumns + ` FROM cms.article WHERE slug = $1 AND deleted_at IS NULL`
	if publishedOnly {
		query += ` AND status = 'published'`
	}
	row := r.conn().QueryRow(ctx, query, slug)
	return scanArticle(row)
}

func (r *CMSRepository) ListArticles(ctx context.Context, status string, limit, offset int32) ([]Article, int, error) {
	where := `deleted_at IS NULL`
	var args []any
	argN := 1
	if status != "" {
		where += ` AND status = $` + strconv.Itoa(argN)
		args = append(args, status)
		argN++
	}
	countQuery := `SELECT COUNT(*) FROM cms.article WHERE ` + where
	var total int
	err := r.conn().QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + articleColumns + ` FROM cms.article WHERE ` + where + ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argN) + ` OFFSET $` + strconv.Itoa(argN+1)
	args = append(args, limit, offset)
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, 0, err
		}
		articles = append(articles, a)
	}
	return articles, total, nil
}

func (r *CMSRepository) PublishArticle(ctx context.Context, id int64) (Article, error) {
	_, err := r.conn().Exec(ctx, `
		UPDATE cms.article
		SET status = 'published', published_at = COALESCE(published_at, NOW()), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return Article{}, err
	}
	return r.GetArticleByID(ctx, id)
}

func (r *CMSRepository) UpsertPage(ctx context.Context, code, slug, title, body string, metaTitle, metaDescription *string) (Page, error) {
	var p Page
	var mt, md pgtype.Text
	if metaTitle != nil {
		mt = pgtype.Text{String: *metaTitle, Valid: true}
	}
	if metaDescription != nil {
		md = pgtype.Text{String: *metaDescription, Valid: true}
	}
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.page (code, slug, title, body, meta_title, meta_description)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (slug) DO UPDATE
		SET title = EXCLUDED.title, body = EXCLUDED.body, meta_title = EXCLUDED.meta_title, meta_description = EXCLUDED.meta_description, updated_at = NOW()
		RETURNING id, code, slug, title, body, meta_title, meta_description, created_at, updated_at
	`, code, slug, title, body, mt, md).Scan(&p.ID, &p.Code, &p.Slug, &p.Title, &p.Body, &mt, &md, &p.CreatedAt, &p.UpdatedAt)
	if mt.Valid {
		s := mt.String
		p.MetaTitle = &s
	}
	if md.Valid {
		s := md.String
		p.MetaDescription = &s
	}
	return p, err
}

func (r *CMSRepository) GetPageBySlug(ctx context.Context, slug string) (Page, error) {
	var p Page
	var mt, md pgtype.Text
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, slug, title, body, meta_title, meta_description, created_at, updated_at
		FROM cms.page WHERE slug = $1 AND deleted_at IS NULL
	`, slug).Scan(&p.ID, &p.Code, &p.Slug, &p.Title, &p.Body, &mt, &md, &p.CreatedAt, &p.UpdatedAt)
	if mt.Valid {
		s := mt.String
		p.MetaTitle = &s
	}
	if md.Valid {
		s := md.String
		p.MetaDescription = &s
	}
	return p, err
}

func (r *CMSRepository) ListPages(ctx context.Context) ([]Page, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, code, slug, title, body, meta_title, meta_description, created_at, updated_at
		FROM cms.page WHERE deleted_at IS NULL ORDER BY title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []Page
	for rows.Next() {
		var p Page
		var mt, md pgtype.Text
		if err := rows.Scan(&p.ID, &p.Code, &p.Slug, &p.Title, &p.Body, &mt, &md, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if mt.Valid {
			s := mt.String
			p.MetaTitle = &s
		}
		if md.Valid {
			s := md.String
			p.MetaDescription = &s
		}
		pages = append(pages, p)
	}
	return pages, nil
}

func (r *CMSRepository) CreateFaq(ctx context.Context, code, question, answer string, category *string, sortOrder int, isActive bool) (Faq, error) {
	var f Faq
	var cat pgtype.Text
	if category != nil {
		cat = pgtype.Text{String: *category, Valid: true}
	}
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.faq (code, question, answer, category, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, code, question, answer, category, sort_order, is_active, created_at, updated_at
	`, code, question, answer, cat, sortOrder, isActive).Scan(&f.ID, &f.Code, &f.Question, &f.Answer, &cat, &f.SortOrder, &f.IsActive, &f.CreatedAt, &f.UpdatedAt)
	if cat.Valid {
		s := cat.String
		f.Category = &s
	}
	return f, err
}

func (r *CMSRepository) GetFaqByCode(ctx context.Context, code string) (Faq, error) {
	var f Faq
	var cat pgtype.Text
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, question, answer, category, sort_order, is_active, created_at, updated_at
		FROM cms.faq WHERE code = $1 AND deleted_at IS NULL
	`, code).Scan(&f.ID, &f.Code, &f.Question, &f.Answer, &cat, &f.SortOrder, &f.IsActive, &f.CreatedAt, &f.UpdatedAt)
	if cat.Valid {
		s := cat.String
		f.Category = &s
	}
	return f, err
}

func (r *CMSRepository) UpdateFaq(ctx context.Context, code, question, answer string, category *string, sortOrder *int, isActive *bool) (Faq, error) {
	var f Faq
	var cat pgtype.Text
	setClauses := []string{}
	args := []any{code}
	argN := 2

	if question != "" {
		setClauses = append(setClauses, fmt.Sprintf("question = $%d", argN))
		args = append(args, question)
		argN++
	}
	if answer != "" {
		setClauses = append(setClauses, fmt.Sprintf("answer = $%d", argN))
		args = append(args, answer)
		argN++
	}
	if category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argN))
		var c pgtype.Text
		if *category != "" {
			c = pgtype.Text{String: *category, Valid: true}
		}
		args = append(args, c)
		argN++
	}
	if sortOrder != nil {
		setClauses = append(setClauses, fmt.Sprintf("sort_order = $%d", argN))
		args = append(args, *sortOrder)
		argN++
	}
	if isActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *isActive)
		argN++
	}

	if len(setClauses) == 0 {
		return r.GetFaqByCode(ctx, code)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(`
		UPDATE cms.faq
		SET %s
		WHERE code = $1 AND deleted_at IS NULL
		RETURNING id, code, question, answer, category, sort_order, is_active, created_at, updated_at
	`, strings.Join(setClauses, ", "))

	err := r.conn().QueryRow(ctx, query, args...).Scan(&f.ID, &f.Code, &f.Question, &f.Answer, &cat, &f.SortOrder, &f.IsActive, &f.CreatedAt, &f.UpdatedAt)
	if cat.Valid {
		s := cat.String
		f.Category = &s
	}
	return f, err
}

func (r *CMSRepository) DeleteFaq(ctx context.Context, code string) error {
	_, err := r.conn().Exec(ctx, `DELETE FROM cms.faq WHERE code = $1`, code)
	return err
}

func (r *CMSRepository) ListFaqsAdmin(ctx context.Context, activeOnly *bool, search string, limit, offset int32) ([]Faq, int, error) {
	where := `deleted_at IS NULL`
	var args []any
	argN := 1

	if activeOnly != nil {
		if *activeOnly {
			where += ` AND is_active = TRUE`
		} else {
			where += ` AND is_active = FALSE`
		}
	}
	if search != "" {
		where += fmt.Sprintf(` AND (question ILIKE $%d OR answer ILIKE $%d OR category ILIKE $%d)`, argN, argN, argN)
		args = append(args, "%"+search+"%")
		argN++
	}

	countQuery := `SELECT COUNT(*) FROM cms.faq WHERE ` + where
	var total int
	err := r.conn().QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, code, question, answer, category, sort_order, is_active, created_at, updated_at
		FROM cms.faq WHERE %s ORDER BY sort_order, category, created_at LIMIT $%d OFFSET $%d
	`, where, argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var faqs []Faq
	for rows.Next() {
		var f Faq
		var cat pgtype.Text
		if err := rows.Scan(&f.ID, &f.Code, &f.Question, &f.Answer, &cat, &f.SortOrder, &f.IsActive, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if cat.Valid {
			s := cat.String
			f.Category = &s
		}
		faqs = append(faqs, f)
	}
	return faqs, total, rows.Err()
}

func (r *CMSRepository) ListFaqs(ctx context.Context, activeOnly bool) ([]Faq, error) {
	faqs, _, err := r.ListFaqsAdmin(ctx, &activeOnly, "", 1000, 0)
	return faqs, err
}

func (r *CMSRepository) CreateBanner(ctx context.Context, code, title, imageUrl string, linkUrl *string, position string, sortOrder int, isActive bool) (Banner, error) {
	var b Banner
	var lu pgtype.Text
	if linkUrl != nil {
		lu = pgtype.Text{String: *linkUrl, Valid: true}
	}
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.banner (code, title, image_url, link_url, position, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, code, title, image_url, link_url, position, sort_order, is_active, created_at
	`, code, title, imageUrl, lu, position, sortOrder, isActive).Scan(&b.ID, &b.Code, &b.Title, &b.ImageUrl, &lu, &b.Position, &b.SortOrder, &b.IsActive, &b.CreatedAt)
	if lu.Valid {
		s := lu.String
		b.LinkUrl = &s
	}
	return b, err
}

func (r *CMSRepository) ListBanners(ctx context.Context, position string) ([]Banner, error) {
	where := `deleted_at IS NULL`
	var args []any
	if position != "" {
		where += ` AND position = $1`
		args = append(args, position)
	}
	query := `SELECT id, code, title, image_url, link_url, position, sort_order, is_active, created_at FROM cms.banner WHERE ` + where + ` ORDER BY sort_order, created_at`
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var banners []Banner
	for rows.Next() {
		var b Banner
		var lu pgtype.Text
		if err := rows.Scan(&b.ID, &b.Code, &b.Title, &b.ImageUrl, &lu, &b.Position, &b.SortOrder, &b.IsActive, &b.CreatedAt); err != nil {
			return nil, err
		}
		if lu.Valid {
			s := lu.String
			b.LinkUrl = &s
		}
		banners = append(banners, b)
	}
	return banners, nil
}

func (r *CMSRepository) UpsertMenuItem(ctx context.Context, code, location, label, url string, parentID *int64, sortOrder int, isActive bool) (MenuItem, error) {
	var m MenuItem
	var pid pgtype.Int8
	if parentID != nil {
		pid = pgtype.Int8{Int64: *parentID, Valid: true}
	}
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.menu_item (code, location, label, url, parent_id, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, code, location, label, url, parent_id, sort_order, is_active, created_at
	`, code, location, label, url, pid, sortOrder, isActive).Scan(&m.ID, &m.Code, &m.Location, &m.Label, &m.Url, &pid, &m.SortOrder, &m.IsActive, &m.CreatedAt)
	if pid.Valid {
		v := pid.Int64
		m.ParentID = &v
	}
	return m, err
}

func (r *CMSRepository) ListMenuItems(ctx context.Context, location string) ([]MenuItem, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, code, location, label, url, parent_id, sort_order, is_active, created_at
		FROM cms.menu_item WHERE location = $1 AND deleted_at IS NULL
		ORDER BY sort_order, label
	`, location)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MenuItem
	for rows.Next() {
		var m MenuItem
		var pid pgtype.Int8
		if err := rows.Scan(&m.ID, &m.Code, &m.Location, &m.Label, &m.Url, &pid, &m.SortOrder, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		if pid.Valid {
			v := pid.Int64
			m.ParentID = &v
		}
		items = append(items, m)
	}
	return items, nil
}

func (r *CMSRepository) SetSetting(ctx context.Context, key, value, groupName string) (Setting, error) {
	var s Setting
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.setting (key, value, group_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, group_name = EXCLUDED.group_name, updated_at = NOW()
		RETURNING key, value, group_name, updated_at
	`, key, value, groupName).Scan(&s.Key, &s.Value, &s.GroupName, &s.UpdatedAt)
	return s, err
}

func (r *CMSRepository) ListSettings(ctx context.Context) ([]Setting, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT key, value, group_name, updated_at FROM cms.setting ORDER BY group_name, key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var settings []Setting
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.Key, &s.Value, &s.GroupName, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, nil
}

type SeoTemplate struct {
	ID                  int64
	Key                 string
	Name                string
	TitleTemplate       string
	DescriptionTemplate *string
	StructuredData      []byte
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type SeoSetting struct {
	ID        int64
	Key       string
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SitemapEntry struct {
	Loc     string
	LastMod time.Time
}

type ProductFeedEntry struct {
	Code         string
	Slug         string
	Name         string
	Description  *string
	SeoTitle     *string
	SeoDesc      *string
	Gtin         *string
	Sku          *string
	PriceMinor   *int64
	Currency     string
	ImageURL     *string
	CategoryName string
	BrandName    *string
	UpdatedAt    time.Time
}

const seoTemplateColumns = `id, key, name, title_template, description_template, structured_data, created_at, updated_at`

func scanSeoTemplate(row interface{ Scan(...any) error }) (SeoTemplate, error) {
	var t SeoTemplate
	var desc pgtype.Text
	var sd []byte
	err := row.Scan(&t.ID, &t.Key, &t.Name, &t.TitleTemplate, &desc, &sd, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	if desc.Valid {
		s := desc.String
		t.DescriptionTemplate = &s
	}
	t.StructuredData = sd
	return t, nil
}

func (r *CMSRepository) CreateSeoTemplate(ctx context.Context, key, name, titleTemplate string, descTemplate *string, structuredData []byte) (SeoTemplate, error) {
	var desc pgtype.Text
	if descTemplate != nil {
		desc = pgtype.Text{String: *descTemplate, Valid: true}
	}
	var id int64
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.seo_template (key, name, title_template, description_template, structured_data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, key, name, titleTemplate, desc, structuredData).Scan(&id)
	if err != nil {
		return SeoTemplate{}, err
	}
	return r.GetSeoTemplateByKey(ctx, key)
}

func (r *CMSRepository) GetSeoTemplateByKey(ctx context.Context, key string) (SeoTemplate, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+seoTemplateColumns+` FROM cms.seo_template WHERE key = $1`, key)
	return scanSeoTemplate(row)
}

func (r *CMSRepository) ListSeoTemplates(ctx context.Context) ([]SeoTemplate, error) {
	rows, err := r.conn().Query(ctx, `SELECT `+seoTemplateColumns+` FROM cms.seo_template ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var templates []SeoTemplate
	for rows.Next() {
		t, err := scanSeoTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func (r *CMSRepository) UpdateSeoTemplate(ctx context.Context, key, name, titleTemplate string, descTemplate *string, structuredData []byte) (SeoTemplate, error) {
	var desc pgtype.Text
	if descTemplate != nil {
		desc = pgtype.Text{String: *descTemplate, Valid: true}
	}
	_, err := r.conn().Exec(ctx, `
		UPDATE cms.seo_template
		SET name = $2, title_template = $3, description_template = $4, structured_data = $5, updated_at = NOW()
		WHERE key = $1
	`, key, name, titleTemplate, desc, structuredData)
	if err != nil {
		return SeoTemplate{}, err
	}
	return r.GetSeoTemplateByKey(ctx, key)
}

func (r *CMSRepository) UpsertSeoSetting(ctx context.Context, key, value string) (SeoSetting, error) {
	var s SeoSetting
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.seo_settings (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
		RETURNING id, key, value, created_at, updated_at
	`, key, value).Scan(&s.ID, &s.Key, &s.Value, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *CMSRepository) GetSeoSettingByKey(ctx context.Context, key string) (SeoSetting, error) {
	var s SeoSetting
	err := r.conn().QueryRow(ctx, `SELECT id, key, value, created_at, updated_at FROM cms.seo_settings WHERE key = $1`, key).Scan(&s.ID, &s.Key, &s.Value, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *CMSRepository) ListSeoSettings(ctx context.Context) ([]SeoSetting, error) {
	rows, err := r.conn().Query(ctx, `SELECT id, key, value, created_at, updated_at FROM cms.seo_settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var settings []SeoSetting
	for rows.Next() {
		var s SeoSetting
		if err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, nil
}

func (r *CMSRepository) ListSitemapEntries(ctx context.Context) ([]SitemapEntry, error) {
	var entries []SitemapEntry

	rows, err := r.conn().Query(ctx, `
		SELECT '/' || slug AS loc, updated_at FROM cms.page WHERE deleted_at IS NULL
		UNION ALL
		SELECT '/articles/' || slug, updated_at FROM cms.article WHERE deleted_at IS NULL AND status = 'published'
		UNION ALL
		SELECT '/products/' || slug, updated_at FROM catalog.product WHERE deleted_at IS NULL AND status = 'published' AND is_active = TRUE
		UNION ALL
		SELECT '/categories/' || slug, updated_at FROM catalog.category WHERE deleted_at IS NULL
		ORDER BY loc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e SitemapEntry
		if err := rows.Scan(&e.Loc, &e.LastMod); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *CMSRepository) ListProductFeedEntries(ctx context.Context) ([]ProductFeedEntry, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT p.code, p.slug, p.name, p.description, p.seo_title, p.seo_description,
		       p.gtin, p.sku, p.base_price_minor, p.currency,
		       (SELECT pm.url FROM catalog.product_media pm WHERE pm.product_id = p.id AND pm.deleted_at IS NULL LIMIT 1),
		       c.name, b.name, p.updated_at
		FROM catalog.product p
		JOIN catalog.category c ON c.id = p.category_id
		LEFT JOIN catalog.brand b ON b.id = p.brand_id
		WHERE p.deleted_at IS NULL AND p.status = 'published' AND p.is_active = TRUE
		ORDER BY p.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []ProductFeedEntry
	for rows.Next() {
		var e ProductFeedEntry
		var desc, seoT, seoD, gtin, sku, imgURL, brand pgtype.Text
		var price pgtype.Int8
		if err := rows.Scan(&e.Code, &e.Slug, &e.Name, &desc, &seoT, &seoD, &gtin, &sku, &price, &e.Currency, &imgURL, &e.CategoryName, &brand, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			s := desc.String
			e.Description = &s
		}
		if seoT.Valid {
			s := seoT.String
			e.SeoTitle = &s
		}
		if seoD.Valid {
			s := seoD.String
			e.SeoDesc = &s
		}
		if gtin.Valid {
			s := gtin.String
			e.Gtin = &s
		}
		if sku.Valid {
			s := sku.String
			e.Sku = &s
		}
		if price.Valid {
			v := price.Int64
			e.PriceMinor = &v
		}
		if imgURL.Valid {
			s := imgURL.String
			e.ImageURL = &s
		}
		if brand.Valid {
			s := brand.String
			e.BrandName = &s
		}
		entries = append(entries, e)
	}
	return entries, nil
}

type HomepageLayout struct {
	ID                int64
	DraftSections     []byte
	PublishedSections []byte
	PublishedAt       *time.Time
	UpdatedAt         time.Time
	UpdatedBy         *string
}

var ErrConcurrentModification = errors.New("concurrent modification detected")

func (r *CMSRepository) GetHomepageLayout(ctx context.Context) (HomepageLayout, error) {
	var h HomepageLayout
	var draft, published []byte
	var pubAt pgtype.Timestamptz
	var updatedBy pgtype.Text
	err := r.conn().QueryRow(ctx, `
		SELECT id, draft_sections, published_sections, published_at, updated_at, updated_by
		FROM cms.homepage_layout WHERE id = 1
	`).Scan(&h.ID, &draft, &published, &pubAt, &h.UpdatedAt, &updatedBy)
	if err != nil {
		return h, err
	}
	h.DraftSections = draft
	h.PublishedSections = published
	if pubAt.Valid {
		t := pubAt.Time
		h.PublishedAt = &t
	}
	if updatedBy.Valid {
		s := updatedBy.String
		h.UpdatedBy = &s
	}
	return h, nil
}

func (r *CMSRepository) GetHomepageLayoutForUpdate(ctx context.Context) (HomepageLayout, error) {
	var h HomepageLayout
	var draft, published []byte
	var pubAt pgtype.Timestamptz
	var updatedBy pgtype.Text
	err := r.conn().QueryRow(ctx, `
		SELECT id, draft_sections, published_sections, published_at, updated_at, updated_by
		FROM cms.homepage_layout WHERE id = 1 FOR UPDATE
	`).Scan(&h.ID, &draft, &published, &pubAt, &h.UpdatedAt, &updatedBy)
	if err != nil {
		return h, err
	}
	h.DraftSections = draft
	h.PublishedSections = published
	if pubAt.Valid {
		t := pubAt.Time
		h.PublishedAt = &t
	}
	if updatedBy.Valid {
		s := updatedBy.String
		h.UpdatedBy = &s
	}
	return h, nil
}

func (r *CMSRepository) UpdateDraftLayout(ctx context.Context, draftSections []byte, updatedBy *string, expectedUpdatedAt *time.Time) (HomepageLayout, error) {
	var ub pgtype.Text
	if updatedBy != nil {
		ub = pgtype.Text{String: *updatedBy, Valid: true}
	}
	if expectedUpdatedAt != nil {
		tag, err := r.conn().Exec(ctx, `
			UPDATE cms.homepage_layout
			SET draft_sections = $1, updated_by = $2, updated_at = NOW()
			WHERE id = 1 AND updated_at = $3
		`, draftSections, ub, *expectedUpdatedAt)
		if err != nil {
			return HomepageLayout{}, err
		}
		if tag.RowsAffected() == 0 {
			return HomepageLayout{}, ErrConcurrentModification
		}
	} else {
		_, err := r.conn().Exec(ctx, `
			INSERT INTO cms.homepage_layout (id, draft_sections, updated_by)
			VALUES (1, $1, $2)
			ON CONFLICT (id) DO UPDATE
			SET draft_sections = $1, updated_by = $2, updated_at = NOW()
		`, draftSections, ub)
		if err != nil {
			return HomepageLayout{}, err
		}
	}
	return r.GetHomepageLayout(ctx)
}

func (r *CMSRepository) PublishHomepage(ctx context.Context, publishedSections []byte, updatedBy *string) (HomepageLayout, error) {
	var ub pgtype.Text
	if updatedBy != nil {
		ub = pgtype.Text{String: *updatedBy, Valid: true}
	}
	_, err := r.conn().Exec(ctx, `
		UPDATE cms.homepage_layout
		SET published_sections = $1, published_at = NOW(), updated_by = $2, updated_at = NOW()
		WHERE id = 1
	`, publishedSections, ub)
	if err != nil {
		return HomepageLayout{}, err
	}
	return r.GetHomepageLayout(ctx)
}

type NewsletterSubscriber struct {
	ID                        int64
	Code                      string
	Email                     string
	Status                    string
	FirstName                 *string
	Source                    *string
	ConfirmationTokenHash     *string
	TokenExpiresAt            *time.Time
	UnsubscribeTokenHash      *string
	UnsubscribeTokenExpiresAt *time.Time
	SubscribedAt              *time.Time
	ConfirmedAt               *time.Time
	UnsubscribedAt            *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

const newsletterSubscriberColumns = `id, code, email, status, first_name, source, confirmation_token_hash, token_expires_at, unsubscribe_token_hash, unsubscribe_token_expires_at, subscribed_at, confirmed_at, unsubscribed_at, created_at, updated_at`

func scanNewsletterSubscriber(row interface{ Scan(...any) error }) (NewsletterSubscriber, error) {
	var s NewsletterSubscriber
	var firstName, source, confirmHash, unsubHash pgtype.Text
	var tokenExp, unsubTokenExp, subscribedAt, confirmedAt, unsubscribedAt pgtype.Timestamptz
	err := row.Scan(
		&s.ID, &s.Code, &s.Email, &s.Status, &firstName, &source,
		&confirmHash, &tokenExp, &unsubHash, &unsubTokenExp,
		&subscribedAt, &confirmedAt, &unsubscribedAt,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return s, err
	}
	if firstName.Valid {
		v := firstName.String
		s.FirstName = &v
	}
	if source.Valid {
		v := source.String
		s.Source = &v
	}
	if confirmHash.Valid {
		v := confirmHash.String
		s.ConfirmationTokenHash = &v
	}
	if tokenExp.Valid {
		t := tokenExp.Time
		s.TokenExpiresAt = &t
	}
	if unsubHash.Valid {
		v := unsubHash.String
		s.UnsubscribeTokenHash = &v
	}
	if unsubTokenExp.Valid {
		t := unsubTokenExp.Time
		s.UnsubscribeTokenExpiresAt = &t
	}
	if subscribedAt.Valid {
		t := subscribedAt.Time
		s.SubscribedAt = &t
	}
	if confirmedAt.Valid {
		t := confirmedAt.Time
		s.ConfirmedAt = &t
	}
	if unsubscribedAt.Valid {
		t := unsubscribedAt.Time
		s.UnsubscribedAt = &t
	}
	return s, nil
}

func (r *CMSRepository) CreateNewsletterSubscriber(ctx context.Context, code, email string, firstName, source *string, confirmTokenHash string, tokenExpiresAt time.Time) (NewsletterSubscriber, error) {
	var fn, src pgtype.Text
	if firstName != nil {
		fn = pgtype.Text{String: *firstName, Valid: true}
	}
	if source != nil {
		src = pgtype.Text{String: *source, Valid: true}
	}
	var id int64
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.newsletter_subscriber (code, email, status, first_name, source, confirmation_token_hash, token_expires_at, subscribed_at)
		VALUES ($1, $2, 'pending', $3, $4, $5, $6, NOW())
		RETURNING id
	`, code, email, fn, src, confirmTokenHash, tokenExpiresAt).Scan(&id)
	if err != nil {
		return NewsletterSubscriber{}, err
	}
	return r.GetNewsletterSubscriberByID(ctx, id)
}

func (r *CMSRepository) GetNewsletterSubscriberByID(ctx context.Context, id int64) (NewsletterSubscriber, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+newsletterSubscriberColumns+` FROM cms.newsletter_subscriber WHERE id = $1`, id)
	return scanNewsletterSubscriber(row)
}

func (r *CMSRepository) GetNewsletterSubscriberByEmail(ctx context.Context, email string) (NewsletterSubscriber, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+newsletterSubscriberColumns+` FROM cms.newsletter_subscriber WHERE email = $1`, email)
	return scanNewsletterSubscriber(row)
}

func (r *CMSRepository) GetNewsletterSubscriberByCode(ctx context.Context, code string) (NewsletterSubscriber, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+newsletterSubscriberColumns+` FROM cms.newsletter_subscriber WHERE code = $1`, code)
	return scanNewsletterSubscriber(row)
}

func (r *CMSRepository) GetNewsletterSubscriberByConfirmTokenHash(ctx context.Context, hash string) (NewsletterSubscriber, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+newsletterSubscriberColumns+` FROM cms.newsletter_subscriber WHERE confirmation_token_hash = $1`, hash)
	return scanNewsletterSubscriber(row)
}

func (r *CMSRepository) GetNewsletterSubscriberByUnsubTokenHash(ctx context.Context, hash string) (NewsletterSubscriber, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+newsletterSubscriberColumns+` FROM cms.newsletter_subscriber WHERE unsubscribe_token_hash = $1`, hash)
	return scanNewsletterSubscriber(row)
}

func (r *CMSRepository) UpdateNewsletterSubscriberStatus(ctx context.Context, id int64, status string, confirmHash *string, tokenExp *time.Time, unsubHash *string, unsubTokenExp *time.Time) (NewsletterSubscriber, error) {
	var ch, uh pgtype.Text
	var te, ute pgtype.Timestamptz
	if confirmHash != nil {
		ch = pgtype.Text{String: *confirmHash, Valid: true}
	}
	if tokenExp != nil {
		te = pgtype.Timestamptz{Time: *tokenExp, Valid: true}
	}
	if unsubHash != nil {
		uh = pgtype.Text{String: *unsubHash, Valid: true}
	}
	if unsubTokenExp != nil {
		ute = pgtype.Timestamptz{Time: *unsubTokenExp, Valid: true}
	}

	setClauses := `status = $2, updated_at = NOW()`
	args := []any{id, status}
	argN := 3

	if confirmHash != nil || ch.Valid {
		setClauses += `, confirmation_token_hash = $` + strconv.Itoa(argN)
		args = append(args, ch)
		argN++
	}
	if tokenExp != nil || te.Valid {
		setClauses += `, token_expires_at = $` + strconv.Itoa(argN)
		args = append(args, te)
		argN++
	}
	if unsubHash != nil || uh.Valid {
		setClauses += `, unsubscribe_token_hash = $` + strconv.Itoa(argN)
		args = append(args, uh)
		argN++
	}
	if unsubTokenExp != nil || ute.Valid {
		setClauses += `, unsubscribe_token_expires_at = $` + strconv.Itoa(argN)
		args = append(args, ute)
		argN++
	}

	switch status {
	case "confirmed":
		setClauses += `, confirmed_at = NOW()`
	case "unsubscribed":
		setClauses += `, unsubscribed_at = NOW()`
	}

	query := `UPDATE cms.newsletter_subscriber SET ` + setClauses + ` WHERE id = $1`
	_, err := r.conn().Exec(ctx, query, args...)
	if err != nil {
		return NewsletterSubscriber{}, err
	}
	return r.GetNewsletterSubscriberByID(ctx, id)
}

func (r *CMSRepository) ListNewsletterSubscribers(ctx context.Context, status, search string, limit, offset int32) ([]NewsletterSubscriber, int, error) {
	where := `1=1`
	var args []any
	argN := 1
	if status != "" {
		where += ` AND status = $` + strconv.Itoa(argN)
		args = append(args, status)
		argN++
	}
	if search != "" {
		where += ` AND (email ILIKE $` + strconv.Itoa(argN) + ` OR first_name ILIKE $` + strconv.Itoa(argN) + `)`
		args = append(args, "%"+search+"%")
		argN++
	}

	countQuery := `SELECT COUNT(*) FROM cms.newsletter_subscriber WHERE ` + where
	var total int
	err := r.conn().QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + newsletterSubscriberColumns + ` FROM cms.newsletter_subscriber WHERE ` + where + ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argN) + ` OFFSET $` + strconv.Itoa(argN+1)
	args = append(args, limit, offset)
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var subs []NewsletterSubscriber
	for rows.Next() {
		s, err := scanNewsletterSubscriber(rows)
		if err != nil {
			return nil, 0, err
		}
		subs = append(subs, s)
	}
	return subs, total, nil
}

func (r *CMSRepository) DeleteNewsletterSubscriber(ctx context.Context, code string) error {
	_, err := r.conn().Exec(ctx, `DELETE FROM cms.newsletter_subscriber WHERE code = $1`, code)
	return err
}

type NewsletterStats struct {
	Total             int
	Pending           int
	Confirmed         int
	Unsubscribed      int
	SubscribedToday   int
	SubscribedLast30D int
}

func (r *CMSRepository) GetNewsletterStats(ctx context.Context) (NewsletterStats, error) {
	var s NewsletterStats
	err := r.conn().QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'pending') AS pending,
			COUNT(*) FILTER (WHERE status = 'confirmed') AS confirmed,
			COUNT(*) FILTER (WHERE status = 'unsubscribed') AS unsubscribed,
			COUNT(*) FILTER (WHERE subscribed_at >= CURRENT_DATE) AS subscribed_today,
			COUNT(*) FILTER (WHERE subscribed_at >= NOW() - INTERVAL '30 days') AS subscribed_last_30_days
		FROM cms.newsletter_subscriber
	`).Scan(&s.Total, &s.Pending, &s.Confirmed, &s.Unsubscribed, &s.SubscribedToday, &s.SubscribedLast30D)
	return s, err
}

func (r *CMSRepository) ListConfirmedSubscribersForExport(ctx context.Context) ([]NewsletterSubscriber, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT `+newsletterSubscriberColumns+`
		FROM cms.newsletter_subscriber
		WHERE status = 'confirmed'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []NewsletterSubscriber
	for rows.Next() {
		s, err := scanNewsletterSubscriber(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

type LegalDocument struct {
	DocType          string
	Title            string
	CurrentVersion   *int
	EffectiveAt      *time.Time
	UpdatedAt        time.Time
	UpdatedBy        *string
	DraftTitle       *string
	DraftBody        *string
	DraftBodyFormat  string
	DraftEffectiveAt *time.Time
	DraftUpdatedAt   *time.Time
	DraftUpdatedBy   *string
}

const legalDocumentColumns = `doc_type, title, current_version, effective_at, updated_at, updated_by, draft_title, draft_body, draft_body_format, draft_effective_at, draft_updated_at, draft_updated_by`

func scanLegalDocument(row interface{ Scan(...any) error }) (LegalDocument, error) {
	var d LegalDocument
	var curVer pgtype.Int4
	var effAt, draftEffAt, draftUpdAt pgtype.Timestamptz
	var updBy, draftTitle, draftBody, draftUpdBy pgtype.Text
	err := row.Scan(
		&d.DocType, &d.Title, &curVer, &effAt, &d.UpdatedAt, &updBy,
		&draftTitle, &draftBody, &d.DraftBodyFormat, &draftEffAt, &draftUpdAt, &draftUpdBy,
	)
	if err != nil {
		return d, err
	}
	if curVer.Valid {
		v := int(curVer.Int32)
		d.CurrentVersion = &v
	}
	if effAt.Valid {
		t := effAt.Time
		d.EffectiveAt = &t
	}
	if updBy.Valid {
		s := updBy.String
		d.UpdatedBy = &s
	}
	if draftTitle.Valid {
		s := draftTitle.String
		d.DraftTitle = &s
	}
	if draftBody.Valid {
		s := draftBody.String
		d.DraftBody = &s
	}
	if draftEffAt.Valid {
		t := draftEffAt.Time
		d.DraftEffectiveAt = &t
	}
	if draftUpdAt.Valid {
		t := draftUpdAt.Time
		d.DraftUpdatedAt = &t
	}
	if draftUpdBy.Valid {
		s := draftUpdBy.String
		d.DraftUpdatedBy = &s
	}
	return d, nil
}

func (r *CMSRepository) CreateLegalDocument(ctx context.Context, docType, title string) (LegalDocument, error) {
	_, err := r.conn().Exec(ctx, `
		INSERT INTO cms.legal_document (doc_type, title)
		VALUES ($1, $2)
	`, docType, title)
	if err != nil {
		return LegalDocument{}, err
	}
	return r.GetLegalDocument(ctx, docType)
}

func (r *CMSRepository) GetLegalDocument(ctx context.Context, docType string) (LegalDocument, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+legalDocumentColumns+` FROM cms.legal_document WHERE doc_type = $1`, docType)
	return scanLegalDocument(row)
}

func (r *CMSRepository) GetLegalDocumentForUpdate(ctx context.Context, docType string) (LegalDocument, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+legalDocumentColumns+` FROM cms.legal_document WHERE doc_type = $1 FOR UPDATE`, docType)
	return scanLegalDocument(row)
}

func (r *CMSRepository) ListLegalDocuments(ctx context.Context, hasDraftFilter *bool, limit, offset int32) ([]LegalDocument, int, error) {
	where := `1=1`
	var args []any
	argN := 1
	if hasDraftFilter != nil {
		if *hasDraftFilter {
			where += ` AND draft_title IS NOT NULL`
		} else {
			where += ` AND draft_title IS NULL`
		}
	}
	countQuery := `SELECT COUNT(*) FROM cms.legal_document WHERE ` + where
	var total int
	err := r.conn().QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	query := `SELECT ` + legalDocumentColumns + ` FROM cms.legal_document WHERE ` + where + ` ORDER BY doc_type ASC LIMIT $` + strconv.Itoa(argN) + ` OFFSET $` + strconv.Itoa(argN+1)
	args = append(args, limit, offset)
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var docs []LegalDocument
	for rows.Next() {
		d, err := scanLegalDocument(rows)
		if err != nil {
			return nil, 0, err
		}
		docs = append(docs, d)
	}
	return docs, total, nil
}

func (r *CMSRepository) UpdateLegalDocumentDraft(ctx context.Context, docType, title, body, bodyFormat string, effectiveAt *time.Time, updatedBy *string, expectedUpdatedAt *time.Time) (LegalDocument, error) {
	var ea pgtype.Timestamptz
	var ubText pgtype.Text
	if effectiveAt != nil {
		ea = pgtype.Timestamptz{Time: *effectiveAt, Valid: true}
	}
	if updatedBy != nil {
		ubText = pgtype.Text{String: *updatedBy, Valid: true}
	}
	if expectedUpdatedAt != nil {
		tag, err := r.conn().Exec(ctx, `
			UPDATE cms.legal_document
			SET draft_title = $2, draft_body = $3, draft_body_format = $4, draft_effective_at = $5, draft_updated_at = NOW(), draft_updated_by = $6, updated_at = NOW(), updated_by = $6
			WHERE doc_type = $1 AND updated_at = $7
		`, docType, title, body, bodyFormat, ea, ubText, *expectedUpdatedAt)
		if err != nil {
			return LegalDocument{}, err
		}
		if tag.RowsAffected() == 0 {
			return LegalDocument{}, ErrConcurrentModification
		}
	} else {
		_, err := r.conn().Exec(ctx, `
			UPDATE cms.legal_document
			SET draft_title = $2, draft_body = $3, draft_body_format = $4, draft_effective_at = $5, draft_updated_at = NOW(), draft_updated_by = $6, updated_at = NOW(), updated_by = $6
			WHERE doc_type = $1
		`, docType, title, body, bodyFormat, ea, ubText)
		if err != nil {
			return LegalDocument{}, err
		}
	}
	return r.GetLegalDocument(ctx, docType)
}

type LegalDocumentVersion struct {
	ID          int64
	DocType     string
	Version     int
	Title       string
	Body        string
	BodyFormat  string
	Status      string
	EffectiveAt time.Time
	PublishedAt time.Time
	CreatedAt   time.Time
	CreatedBy   *string
}

const legalDocumentVersionColumns = `id, doc_type, version, title, body, body_format, status, effective_at, published_at, created_at, created_by`

func scanLegalDocumentVersion(row interface{ Scan(...any) error }) (LegalDocumentVersion, error) {
	var v LegalDocumentVersion
	var createdBy pgtype.Text
	err := row.Scan(
		&v.ID, &v.DocType, &v.Version, &v.Title, &v.Body, &v.BodyFormat,
		&v.Status, &v.EffectiveAt, &v.PublishedAt, &v.CreatedAt, &createdBy,
	)
	if err != nil {
		return v, err
	}
	if createdBy.Valid {
		s := createdBy.String
		v.CreatedBy = &s
	}
	return v, nil
}

func (r *CMSRepository) GetNextVersion(ctx context.Context, docType string) (int, error) {
	var maxVer pgtype.Int4
	err := r.conn().QueryRow(ctx, `SELECT MAX(version) FROM cms.legal_document_version WHERE doc_type = $1`, docType).Scan(&maxVer)
	if err != nil {
		return 0, err
	}
	if maxVer.Valid {
		return int(maxVer.Int32) + 1, nil
	}
	return 1, nil
}

func (r *CMSRepository) PublishLegalDocumentVersion(ctx context.Context, docType string, version int, title, body, bodyFormat string, effectiveAt time.Time, createdBy *string) (LegalDocumentVersion, error) {
	var cb pgtype.Text
	if createdBy != nil {
		cb = pgtype.Text{String: *createdBy, Valid: true}
	}
	var id int64
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.legal_document_version (doc_type, version, title, body, body_format, status, effective_at, published_at, created_by)
		VALUES ($1, $2, $3, $4, $5, 'published', $6, NOW(), $7)
		RETURNING id
	`, docType, version, title, body, bodyFormat, effectiveAt, cb).Scan(&id)
	if err != nil {
		return LegalDocumentVersion{}, err
	}
	return r.GetLegalDocumentVersion(ctx, id)
}

func (r *CMSRepository) SupersedePreviousVersion(ctx context.Context, docType string, excludeVersion int) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE cms.legal_document_version
		SET status = 'superseded'
		WHERE doc_type = $1 AND status = 'published' AND version != $2
	`, docType, excludeVersion)
	return err
}

func (r *CMSRepository) UpdateLegalDocumentCurrentVersion(ctx context.Context, docType string, version int, effectiveAt time.Time, updatedBy *string) error {
	var ub pgtype.Text
	if updatedBy != nil {
		ub = pgtype.Text{String: *updatedBy, Valid: true}
	}
	_, err := r.conn().Exec(ctx, `
		UPDATE cms.legal_document
		SET current_version = $2, effective_at = $3, updated_at = NOW(), updated_by = $4,
		    draft_title = NULL, draft_body = NULL, draft_body_format = 'markdown', draft_effective_at = NULL, draft_updated_at = NULL, draft_updated_by = NULL
		WHERE doc_type = $1
	`, docType, version, effectiveAt, ub)
	return err
}

func (r *CMSRepository) GetLegalDocumentVersion(ctx context.Context, id int64) (LegalDocumentVersion, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+legalDocumentVersionColumns+` FROM cms.legal_document_version WHERE id = $1`, id)
	return scanLegalDocumentVersion(row)
}

func (r *CMSRepository) GetLegalDocumentVersionByNumber(ctx context.Context, docType string, version int) (LegalDocumentVersion, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+legalDocumentVersionColumns+` FROM cms.legal_document_version WHERE doc_type = $1 AND version = $2`, docType, version)
	return scanLegalDocumentVersion(row)
}

func (r *CMSRepository) GetCurrentEffectiveVersion(ctx context.Context, docType string) (LegalDocumentVersion, error) {
	row := r.conn().QueryRow(ctx, `
		SELECT `+legalDocumentVersionColumns+`
		FROM cms.legal_document_version
		WHERE doc_type = $1 AND status = 'published' AND effective_at <= NOW()
		ORDER BY version DESC
		LIMIT 1
	`, docType)
	return scanLegalDocumentVersion(row)
}

func (r *CMSRepository) ListLegalDocumentVersions(ctx context.Context, docType, status string, limit, offset int32) ([]LegalDocumentVersion, int, error) {
	where := `doc_type = $1`
	args := []any{docType}
	argN := 2
	if status != "" {
		where += ` AND status = $` + strconv.Itoa(argN)
		args = append(args, status)
		argN++
	}
	countQuery := `SELECT COUNT(*) FROM cms.legal_document_version WHERE ` + where
	var total int
	err := r.conn().QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	query := `SELECT ` + legalDocumentVersionColumns + ` FROM cms.legal_document_version WHERE ` + where + ` ORDER BY version DESC LIMIT $` + strconv.Itoa(argN) + ` OFFSET $` + strconv.Itoa(argN+1)
	args = append(args, limit, offset)
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var versions []LegalDocumentVersion
	for rows.Next() {
		v, err := scanLegalDocumentVersion(rows)
		if err != nil {
			return nil, 0, err
		}
		versions = append(versions, v)
	}
	return versions, total, nil
}

func (r *CMSRepository) GetLatestPublishedVersion(ctx context.Context, docType string) (LegalDocumentVersion, error) {
	row := r.conn().QueryRow(ctx, `
		SELECT `+legalDocumentVersionColumns+`
		FROM cms.legal_document_version
		WHERE doc_type = $1 AND status = 'published'
		ORDER BY version DESC
		LIMIT 1
	`, docType)
	return scanLegalDocumentVersion(row)
}

type ContactEnquiry struct {
	ID        int64
	Code      string
	Name      string
	Email     string
	Phone     *string
	Company   *string
	Subject   string
	Message   string
	Source    string
	Status    string
	LeadID    *int64
	IPAddress *string
	UserAgent *string
	CreatedAt time.Time
	UpdatedAt time.Time
	RoutedAt  *time.Time
}

const contactEnquiryColumns = `id, code, name, email, phone, company, subject, message, source, status, lead_id, ip_address, user_agent, created_at, updated_at, routed_at`

func scanContactEnquiry(row interface{ Scan(...any) error }) (ContactEnquiry, error) {
	var e ContactEnquiry
	var phone, company, ipAddr, userAgent pgtype.Text
	var leadID pgtype.Int8
	var routedAt pgtype.Timestamptz
	err := row.Scan(
		&e.ID, &e.Code, &e.Name, &e.Email, &phone, &company, &e.Subject, &e.Message,
		&e.Source, &e.Status, &leadID, &ipAddr, &userAgent,
		&e.CreatedAt, &e.UpdatedAt, &routedAt,
	)
	if err != nil {
		return e, err
	}
	if phone.Valid {
		s := phone.String
		e.Phone = &s
	}
	if company.Valid {
		s := company.String
		e.Company = &s
	}
	if leadID.Valid {
		v := leadID.Int64
		e.LeadID = &v
	}
	if ipAddr.Valid {
		s := ipAddr.String
		e.IPAddress = &s
	}
	if userAgent.Valid {
		s := userAgent.String
		e.UserAgent = &s
	}
	if routedAt.Valid {
		t := routedAt.Time
		e.RoutedAt = &t
	}
	return e, nil
}

func (r *CMSRepository) CreateContactEnquiry(ctx context.Context, code, name, email string, phone, company *string, subject, message, source string, ipAddress, userAgent *string) (ContactEnquiry, error) {
	var ph, co, ip, ua pgtype.Text
	if phone != nil {
		ph = pgtype.Text{String: *phone, Valid: true}
	}
	if company != nil {
		co = pgtype.Text{String: *company, Valid: true}
	}
	if ipAddress != nil {
		ip = pgtype.Text{String: *ipAddress, Valid: true}
	}
	if userAgent != nil {
		ua = pgtype.Text{String: *userAgent, Valid: true}
	}
	var id int64
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.contact_enquiry (code, name, email, phone, company, subject, message, source, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, code, name, email, ph, co, subject, message, source, ip, ua).Scan(&id)
	if err != nil {
		return ContactEnquiry{}, err
	}
	return r.GetContactEnquiryByID(ctx, id)
}

func (r *CMSRepository) GetContactEnquiryByID(ctx context.Context, id int64) (ContactEnquiry, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+contactEnquiryColumns+` FROM cms.contact_enquiry WHERE id = $1`, id)
	return scanContactEnquiry(row)
}

func (r *CMSRepository) GetContactEnquiryByCode(ctx context.Context, code string) (ContactEnquiry, error) {
	row := r.conn().QueryRow(ctx, `SELECT `+contactEnquiryColumns+` FROM cms.contact_enquiry WHERE code = $1`, code)
	return scanContactEnquiry(row)
}

func (r *CMSRepository) ListContactEnquiries(ctx context.Context, status, source, search string, dateFrom, dateTo *time.Time, limit, offset int32) ([]ContactEnquiry, int, error) {
	where := `1=1`
	var args []any
	argN := 1
	if status != "" {
		where += ` AND status = $` + strconv.Itoa(argN)
		args = append(args, status)
		argN++
	}
	if source != "" {
		where += ` AND source = $` + strconv.Itoa(argN)
		args = append(args, source)
		argN++
	}
	if search != "" {
		where += ` AND (name ILIKE $` + strconv.Itoa(argN) + ` OR email ILIKE $` + strconv.Itoa(argN) + ` OR company ILIKE $` + strconv.Itoa(argN) + ` OR subject ILIKE $` + strconv.Itoa(argN) + `)`
		args = append(args, "%"+search+"%")
		argN++
	}
	if dateFrom != nil {
		where += ` AND created_at >= $` + strconv.Itoa(argN)
		args = append(args, *dateFrom)
		argN++
	}
	if dateTo != nil {
		where += ` AND created_at <= $` + strconv.Itoa(argN)
		args = append(args, *dateTo)
		argN++
	}

	countQuery := `SELECT COUNT(*) FROM cms.contact_enquiry WHERE ` + where
	var total int
	err := r.conn().QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + contactEnquiryColumns + ` FROM cms.contact_enquiry WHERE ` + where + ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argN) + ` OFFSET $` + strconv.Itoa(argN+1)
	args = append(args, limit, offset)
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var enquiries []ContactEnquiry
	for rows.Next() {
		e, err := scanContactEnquiry(rows)
		if err != nil {
			return nil, 0, err
		}
		enquiries = append(enquiries, e)
	}
	return enquiries, total, nil
}

func (r *CMSRepository) UpdateContactEnquiryLeadID(ctx context.Context, id int64, leadID int64) (ContactEnquiry, error) {
	_, err := r.conn().Exec(ctx, `
		UPDATE cms.contact_enquiry
		SET lead_id = $2, routed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id, leadID)
	if err != nil {
		return ContactEnquiry{}, err
	}
	return r.GetContactEnquiryByID(ctx, id)
}
