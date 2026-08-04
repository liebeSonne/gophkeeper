package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
)

type FileRepo struct {
	pool *pgxpool.Pool
}

func NewFileRepo(pool *pgxpool.Pool) *FileRepo {
	return &FileRepo{pool: pool}
}

func (r *FileRepo) NextID(_ context.Context) uuid.UUID {
	return uuid.New()
}

func (r *FileRepo) StoreFile(ctx context.Context, file model.File) error {
	const query = `
		INSERT INTO file (id, user_id, name, mime_type, size, chunks_count, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query, file.ID, file.UserID, file.Name, file.MimeType, file.Size, file.ChunksCount, file.Status, file.CreatedAt, file.UpdatedAt)
	if err != nil {
		return fmt.Errorf("store file: %w", err)
	}
	return nil
}

func (r *FileRepo) UpdateFileStatus(ctx context.Context, id uuid.UUID, status model.FileStatus) error {
	const query = `
		UPDATE file SET status = $1, updated_at = NOW() WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update file status: %w", err)
	}
	return nil
}

func (r *FileRepo) GetFileByID(ctx context.Context, id uuid.UUID) (model.File, error) {
	const query = `
		SELECT id, user_id, name, mime_type, size, chunks_count, status, created_at, updated_at
		FROM file
		WHERE id = $1
	`
	file := model.File{}
	err := r.pool.QueryRow(ctx, query, id).Scan(&file.ID, &file.UserID, &file.Name, &file.MimeType, &file.Size, &file.ChunksCount, &file.Status, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.File{}, repository.ErrNotFound
		}
		return model.File{}, fmt.Errorf("get file by id: %w", err)
	}
	return file, nil
}

func (r *FileRepo) DeleteFile(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM file WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func (r *FileRepo) StoreFileChunks(ctx context.Context, chunks []model.FileChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	const query = `
		INSERT INTO file_chunk (id, file_id, chunk_index, uploaded, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (file_id, chunk_index) DO UPDATE SET
			uploaded = EXCLUDED.uploaded
	`

	for _, chunk := range chunks {
		_, err := r.pool.Exec(ctx, query, chunk.ID, chunk.FileID, chunk.ChunkIndex, chunk.Uploaded, chunk.CreatedAt)
		if err != nil {
			return fmt.Errorf("store file chunk: %w", err)
		}
	}
	return nil
}

func (r *FileRepo) UpdateChunkUploaded(ctx context.Context, fileID uuid.UUID, chunkIndex int, uploaded bool) error {
	const query = `
		UPDATE file_chunk SET uploaded = $1 
	  	WHERE file_id = $2 AND chunk_index = $3
	`
	result, err := r.pool.Exec(ctx, query, uploaded, fileID, chunkIndex)
	if err != nil {
		return fmt.Errorf("update chunk uploaded: %w", err)
	}
	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *FileRepo) GetUploadedChunksCount(ctx context.Context, fileID uuid.UUID) (int, error) {
	const query = `
		SELECT count(*) 
		FROM file_chunk 
		WHERE file_id = $1 AND uploaded = TRUE
	`
	var count int
	err := r.pool.QueryRow(ctx, query, fileID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count uploaded chunks: %w", err)
	}
	return count, nil
}

func (r *FileRepo) GetExistingFilesByUserID(ctx context.Context, userID uuid.UUID, fileIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(fileIDs) == 0 {
		return []uuid.UUID{}, nil
	}

	const query = `
		SELECT id 
		FROM file 
		WHERE id = ANY($1) AND user_id = $2
	`
	rows, err := r.pool.Query(ctx, query, fileIDs, userID)
	if err != nil {
		return nil, fmt.Errorf("get existing files by user id: %w", err)
	}
	defer rows.Close()

	result := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		err = rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("scan file id: %w", err)
		}
		result = append(result, id)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate files: %w", err)
	}
	return result, nil
}

func (r *FileRepo) ListByUserID(ctx context.Context, userID uuid.UUID, query *string, limit, offset *int) ([]model.File, error) {
	baseQuery := `
		SELECT id, user_id, name, mime_type, size, chunks_count, status, created_at, updated_at 
		FROM file 
		WHERE %s
		ORDER BY created_at DESC
	`

	conditions, args := r.prepareConditions(userID, query)

	sqlQuery := fmt.Sprintf(baseQuery, strings.Join(conditions, " AND "))
	if limit != nil {
		sqlQuery += fmt.Sprintf(" LIMIT %d", *limit)
	}
	if offset != nil {
		sqlQuery += fmt.Sprintf(" OFFSET %d", *offset)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list files by user id: %w", err)
	}
	defer rows.Close()

	files := make([]model.File, 0)
	for rows.Next() {
		var f model.File
		err = rows.Scan(&f.ID, &f.UserID, &f.Name, &f.MimeType, &f.Size, &f.ChunksCount, &f.Status, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate files: %w", err)
	}
	return files, nil
}

func (r *FileRepo) CountByUserID(ctx context.Context, userID uuid.UUID, query *string) (int, error) {
	sqlQuery := `SELECT count(*) FROM file WHERE %s`

	conditions, args := r.prepareConditions(userID, query)

	var count int
	err := r.pool.QueryRow(ctx, fmt.Sprintf(sqlQuery, strings.Join(conditions, " AND ")), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count files by user id: %w", err)
	}
	return count, nil
}

func (r *FileRepo) prepareConditions(userID uuid.UUID, query *string) (conditions []string, args []interface{}) {
	conditions = []string{
		"user_id = $1",
	}
	args = []interface{}{
		userID,
	}

	argIndex := 1
	if query != nil && *query != "" {
		argIndex++
		conditions = append(conditions, "name ILIKE $"+fmt.Sprint(argIndex))
		args = append(args, "%"+*query+"%")
	}

	return conditions, args
}
