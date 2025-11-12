// repository/faq_repository.go
package repository

import (
	"context"
	"errors"
	"fmt"
	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FAQRepository struct {
	db *pgxpool.Pool
}

func NewFAQRepository(db *pgxpool.Pool) *FAQRepository {
	return &FAQRepository{db: db}
}

// Create создает новый FAQ
func (r *FAQRepository) Create(ctx context.Context, ext RepoExtension, faq *model.FAQ) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
		INSERT INTO sso.faqs (id, question, answer, category, "order", is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	now := time.Now()
	faq.CreatedAt = now
	faq.UpdatedAt = now

	err := ext.QueryRow(ctx, query,
		faq.ID,
		faq.Question,
		faq.Answer,
		faq.Category,
		faq.Order,
		faq.IsActive,
		faq.CreatedBy,
		faq.CreatedAt,
		faq.UpdatedAt,
	).Scan(&faq.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return apperrors.ErrFAQAlreadyExists
			}
		}
		return fmt.Errorf("failed to create FAQ: %w", err)
	}

	return nil
}

// GetByID возвращает FAQ по ID
func (r *FAQRepository) GetByID(ctx context.Context, ext RepoExtension, id uuid.UUID) (*model.FAQ, error) {
	if ext == nil {
		ext = r.db
	}

	const query = `
		SELECT id, question, answer, category, "order", is_active, created_by, created_at, updated_at
		FROM sso.faqs
		WHERE id = $1 AND deleted_at IS NULL
	`

	var faq model.FAQ
	err := ext.QueryRow(ctx, query, id).Scan(
		&faq.ID,
		&faq.Question,
		&faq.Answer,
		&faq.Category,
		&faq.Order,
		&faq.IsActive,
		&faq.CreatedBy,
		&faq.CreatedAt,
		&faq.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrFAQNotFound
		}
		return nil, fmt.Errorf("failed to get FAQ by ID: %w", err)
	}

	return &faq, nil
}

// Update обновляет FAQ
func (r *FAQRepository) Update(ctx context.Context, ext RepoExtension, id uuid.UUID, updateData *model.FAQUpdateRequest) error {
	if ext == nil {
		ext = r.db
	}

	// Динамическое построение запроса
	query := "UPDATE sso.faqs SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2

	if updateData.Question != nil {
		query += fmt.Sprintf(", question = $%d", argIndex)
		args = append(args, *updateData.Question)
		argIndex++
	}

	if updateData.Answer != nil {
		query += fmt.Sprintf(", answer = $%d", argIndex)
		args = append(args, *updateData.Answer)
		argIndex++
	}

	if updateData.Category != nil {
		query += fmt.Sprintf(", category = $%d", argIndex)
		args = append(args, *updateData.Category)
		argIndex++
	}

	if updateData.Order != nil {
		query += fmt.Sprintf(", \"order\" = $%d", argIndex)
		args = append(args, *updateData.Order)
		argIndex++
	}

	if updateData.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIndex)
		args = append(args, *updateData.IsActive)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND deleted_at IS NULL", argIndex)
	args = append(args, id)

	result, err := ext.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update FAQ: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrFAQNotFound
	}

	return nil
}

// Delete мягко удаляет FAQ
func (r *FAQRepository) Delete(ctx context.Context, ext RepoExtension, id uuid.UUID) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
		UPDATE sso.faqs 
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := ext.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete FAQ: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrFAQNotFound
	}

	return nil
}

// List возвращает список FAQ с фильтрацией
func (r *FAQRepository) List(ctx context.Context, ext RepoExtension, params model.FAQQueryParams) ([]model.FAQ, int, error) {
	if ext == nil {
		ext = r.db
	}

	// Базовый запрос
	baseQuery := `
		FROM sso.faqs 
		WHERE deleted_at IS NULL
	`
	countQuery := "SELECT COUNT(*) " + baseQuery
	selectQuery := `
		SELECT id, question, answer, category, "order", is_active, created_by, created_at, updated_at 
	` + baseQuery

	args := []interface{}{}
	argIndex := 1

	// Добавляем условия фильтрации
	if params.Category != "" {
		baseQuery += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, params.Category)
		argIndex++
	}

	if params.IsActive != nil {
		baseQuery += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *params.IsActive)
		argIndex++
	}

	// Получаем общее количество
	var total int
	err := ext.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count FAQs: %w", err)
	}

	// Добавляем сортировку и пагинацию
	selectQuery += " ORDER BY \"order\" ASC, created_at DESC"

	if params.Limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, params.Limit)
		argIndex++
	}

	if params.Offset > 0 {
		selectQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, params.Offset)
	}

	// Выполняем запрос на получение данных
	rows, err := ext.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list FAQs: %w", err)
	}
	defer rows.Close()

	var faqs []model.FAQ
	for rows.Next() {
		var faq model.FAQ
		err := rows.Scan(
			&faq.ID,
			&faq.Question,
			&faq.Answer,
			&faq.Category,
			&faq.Order,
			&faq.IsActive,
			&faq.CreatedBy,
			&faq.CreatedAt,
			&faq.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan FAQ: %w", err)
		}
		faqs = append(faqs, faq)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating FAQ rows: %w", err)
	}

	return faqs, total, nil
}

// GetByCategory возвращает FAQ по категории
func (r *FAQRepository) GetByCategory(ctx context.Context, ext RepoExtension, category string) ([]model.FAQ, error) {
	if ext == nil {
		ext = r.db
	}

	const query = `
		SELECT id, question, answer, category, "order", is_active, created_by, created_at, updated_at
		FROM sso.faqs
		WHERE category = $1 AND is_active = true AND deleted_at IS NULL
		ORDER BY "order" ASC, created_at DESC
	`

	rows, err := ext.Query(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get FAQs by category: %w", err)
	}
	defer rows.Close()

	var faqs []model.FAQ
	for rows.Next() {
		var faq model.FAQ
		err := rows.Scan(
			&faq.ID,
			&faq.Question,
			&faq.Answer,
			&faq.Category,
			&faq.Order,
			&faq.IsActive,
			&faq.CreatedBy,
			&faq.CreatedAt,
			&faq.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan FAQ: %w", err)
		}
		faqs = append(faqs, faq)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating FAQ rows: %w", err)
	}

	return faqs, nil
}

// GetCategories возвращает список всех категорий
func (r *FAQRepository) GetCategories(ctx context.Context, ext RepoExtension) ([]string, error) {
	if ext == nil {
		ext = r.db
	}

	const query = `
		SELECT DISTINCT category
		FROM sso.faqs
		WHERE is_active = true AND deleted_at IS NULL
		ORDER BY category
	`

	rows, err := ext.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	return categories, nil
}
