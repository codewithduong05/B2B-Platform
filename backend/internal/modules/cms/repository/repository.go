package repository

import (
	"context"
	"strconv"
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
	SortOrder int
	IsActive  bool
	CreatedAt time.Time
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

func (r *CMSRepository) CreateFaq(ctx context.Context, code, question, answer string, sortOrder int, isActive bool) (Faq, error) {
	var f Faq
	err := r.conn().QueryRow(ctx, `
		INSERT INTO cms.faq (code, question, answer, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, code, question, answer, sort_order, is_active, created_at
	`, code, question, answer, sortOrder, isActive).Scan(&f.ID, &f.Code, &f.Question, &f.Answer, &f.SortOrder, &f.IsActive, &f.CreatedAt)
	return f, err
}

func (r *CMSRepository) ListFaqs(ctx context.Context, activeOnly bool) ([]Faq, error) {
	query := `SELECT id, code, question, answer, sort_order, is_active, created_at FROM cms.faq WHERE deleted_at IS NULL`
	if activeOnly {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY sort_order, created_at`
	rows, err := r.conn().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var faqs []Faq
	for rows.Next() {
		var f Faq
		if err := rows.Scan(&f.ID, &f.Code, &f.Question, &f.Answer, &f.SortOrder, &f.IsActive, &f.CreatedAt); err != nil {
			return nil, err
		}
		faqs = append(faqs, f)
	}
	return faqs, nil
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
