package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
)

type DataRepo struct {
	pool   *pgxpool.Pool
	logger intlogger.Logger
}

func NewDataRepo(
	pool *pgxpool.Pool,
	logger intlogger.Logger,
) *DataRepo {
	return &DataRepo{
		pool:   pool,
		logger: logger,
	}
}

type ListSpec struct {
	UserID    uuid.UUID
	Query     *string
	DataTypes []model.DataType
	Limit     *int
	Offset    *int
}

func (r *DataRepo) NextID(_ context.Context) uuid.UUID {
	return uuid.New()
}

func (r *DataRepo) Store(ctx context.Context, data model.Data) error {
	const query = `
		INSERT INTO data (id, user_id, type, payload, metadata, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		ON CONFLICT (id) DO UPDATE SET 
			type = EXCLUDED.type, 
			payload = EXCLUDED.payload, 
			metadata = EXCLUDED.metadata, 
			updated_at = EXCLUDED.updated_at
		`
	_, err := r.pool.Exec(ctx, query, data.ID, data.UserID, data.Type, data.Payload, data.Metadata, data.CreatedAt, data.UpdatedAt)
	if err != nil {
		return fmt.Errorf("store data: %w", err)
	}
	return nil
}

func (r *DataRepo) GetByID(ctx context.Context, id uuid.UUID) (model.Data, error) {
	const query = `
		SELECT id, user_id, type, payload, metadata, created_at, updated_at 
		FROM data 
		WHERE id = $1
	`
	data := model.Data{}
	err := r.pool.QueryRow(ctx, query, id).Scan(&data.ID, &data.UserID, &data.Type, &data.Payload, &data.Metadata, &data.CreatedAt, &data.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Data{}, repository.ErrNotFound
		}
		return model.Data{}, fmt.Errorf("get data by id: %w", err)
	}
	return data, nil
}

func (r *DataRepo) Delete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	const query = `DELETE FROM data WHERE id = ANY($1)`
	_, err := r.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("delete data: %w", err)
	}
	return nil
}

func (r *DataRepo) ListWithCount(ctx context.Context, spec ListSpec) ([]model.Data, int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx for list with count: %w", err)
	}

	defer func() {
		errRollback := tx.Rollback(ctx)
		if errRollback != nil && !errors.Is(errRollback, pgx.ErrTxClosed) && !errors.Is(errRollback, sql.ErrTxDone) {
			r.logger.Error("error on rollback transaction", "err", errRollback)
		}
	}()

	count, err := r.countInTx(ctx, tx, spec)
	if err != nil {
		return nil, 0, err
	}

	items, err := r.listInTx(ctx, tx, spec)
	if err != nil {
		return nil, 0, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("commit tx for list with count: %w", err)
	}

	return items, count, nil
}

func (r *DataRepo) countInTx(ctx context.Context, tx pgx.Tx, spec ListSpec) (int, error) {
	const query = `SELECT count(*) FROM data WHERE %s`

	conditions, args := r.prepareSpecConditions(spec)

	var count int
	err := tx.QueryRow(ctx, fmt.Sprintf(query, strings.Join(conditions, " AND ")), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count data: %w", err)
	}
	return count, nil
}

func (r *DataRepo) listInTx(ctx context.Context, tx pgx.Tx, spec ListSpec) ([]model.Data, error) {
	baseQuery := `
		SELECT id, user_id, type, payload, metadata, created_at, updated_at
		FROM data
		WHERE %s
		ORDER BY created_at DESC
	`

	conditions, args := r.prepareSpecConditions(spec)

	query := fmt.Sprintf(baseQuery, strings.Join(conditions, " AND "))
	if spec.Limit != nil {
		query += fmt.Sprintf(" LIMIT %d", *spec.Limit)
	}
	if spec.Offset != nil {
		query += fmt.Sprintf(" OFFSET %d", *spec.Offset)
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list data: %w", err)
	}
	defer rows.Close()

	var result []model.Data
	for rows.Next() {
		var data model.Data
		err = rows.Scan(&data.ID, &data.UserID, &data.Type, &data.Payload, &data.Metadata, &data.CreatedAt, &data.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan data row: %w", err)
		}
		result = append(result, data)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate data rows: %w", err)
	}
	return result, nil
}

func (r *DataRepo) prepareSpecConditions(spec ListSpec) (conditions []string, args []interface{}) {
	conditions = []string{
		"user_id = $1",
	}
	args = []interface{}{
		spec.UserID,
	}

	argIndex := 1
	if spec.Query != nil && *spec.Query != "" {
		argIndex++
		conditions = append(conditions, "metadata ILIKE $"+fmt.Sprint(argIndex))
		args = append(args, "%"+*spec.Query+"%")
	}
	if len(spec.DataTypes) > 0 {
		argIndex++
		conditions = append(conditions, "type = ANY($"+fmt.Sprint(argIndex)+")")
		args = append(args, spec.DataTypes)
	}

	return conditions, args
}
