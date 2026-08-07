package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/liebeSonne/gophkeeper/internal/crypto"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
	"github.com/liebeSonne/gophkeeper/internal/storage"
)

const (
	minioChunksPrefix = "chunks/"
	minioFilesPrefix  = "files/"
)

func NewFileService(
	fileRepo FileRepository,
	encryptor crypto.Encryptor,
	minio storage.MinIOClient,
	bucket string,
	logger intlogger.Logger,
) *FileService {
	return &FileService{
		fileRepo:  fileRepo,
		encryptor: encryptor,
		minio:     minio,
		bucket:    bucket,
		logger:    logger,
	}
}

type FileService struct {
	fileRepo  FileRepository
	encryptor crypto.Encryptor
	minio     storage.MinIOClient
	bucket    string
	logger    intlogger.Logger
}

func (s *FileService) InitUpload(
	ctx context.Context,
	userID uuid.UUID,
	name string,
	mimeType string,
	size int64,
	chunksCount int,
) (model.File, error) {
	if chunksCount < 1 {
		return model.File{}, ErrInvalidChunksCount
	}

	now := time.Now()
	file := model.File{
		ID:          s.fileRepo.NextID(ctx),
		UserID:      userID,
		Name:        name,
		MimeType:    mimeType,
		Size:        size,
		ChunksCount: chunksCount,
		Status:      model.FileStatusInProgress,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := s.fileRepo.StoreFile(ctx, file)
	if err != nil {
		return model.File{}, fmt.Errorf("store file: %w", err)
	}

	chunks := make([]model.FileChunk, 0, chunksCount)
	for i := 0; i < chunksCount; i++ {
		chunks = append(chunks, model.FileChunk{
			ID:         s.fileRepo.NextID(ctx),
			FileID:     file.ID,
			ChunkIndex: i,
			Uploaded:   false,
			CreatedAt:  now,
		})
	}

	err = s.fileRepo.StoreFileChunks(ctx, chunks)
	if err != nil {
		return model.File{}, fmt.Errorf("store file chunks: %w", err)
	}

	return file, nil
}

func (s *FileService) UploadChunk(
	ctx context.Context,
	fileID uuid.UUID,
	userID uuid.UUID,
	chunkIndex int,
	data []byte,
) error {
	file, err := s.fileRepo.GetFileByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrFileNotFound
		}
		return fmt.Errorf("get file: %w", err)
	}

	if file.UserID != userID {
		return ErrFileAccessDenied
	}

	if file.Status != model.FileStatusInProgress {
		return ErrFileUploadNotInProgress
	}

	if chunkIndex < 0 || chunkIndex >= file.ChunksCount {
		return ErrInvalidChunkIndex
	}

	encrypted, err := s.encryptor.Encrypt(data)
	if err != nil {
		return fmt.Errorf("encrypt chunk: %w", err)
	}

	objectKey := s.makeFileChunkKey(fileID, chunkIndex)
	err = s.minio.PutObject(ctx, s.bucket, objectKey, bytes.NewReader(encrypted), int64(len(encrypted)))
	if err != nil {
		return fmt.Errorf("upload chunk to storage: %w", err)
	}

	err = s.fileRepo.UpdateChunkUploaded(ctx, fileID, chunkIndex, true)
	if err != nil {
		return fmt.Errorf("update chunk status: %w", err)
	}

	return nil
}

func (s *FileService) CompleteUpload(
	ctx context.Context,
	fileID uuid.UUID,
	userID uuid.UUID,
) (model.File, error) {
	file, err := s.fileRepo.GetFileByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.File{}, ErrFileNotFound
		}
		return model.File{}, fmt.Errorf("get file: %w", err)
	}

	if file.UserID != userID {
		return model.File{}, ErrFileAccessDenied
	}

	if file.Status != model.FileStatusInProgress {
		return model.File{}, ErrFileUploadNotInProgress
	}

	uploadedCount, err := s.fileRepo.GetUploadedChunksCount(ctx, fileID)
	if err != nil {
		return model.File{}, fmt.Errorf("count uploaded chunks: %w", err)
	}

	if uploadedCount != file.ChunksCount {
		return model.File{}, fmt.Errorf("not all chunks uploaded: %d/%d", uploadedCount, file.ChunksCount)
	}

	objectKeys := make([]string, 0, file.ChunksCount)
	for i := 0; i < file.ChunksCount; i++ {
		objectKey := s.makeFileChunkKey(fileID, i)
		objectKeys = append(objectKeys, objectKey)
	}

	composedKey := s.makeFileKey(fileID)

	var readers []io.ReadCloser
	defer func() {
		for _, r := range readers {
			errClose := r.Close()
			if errClose != nil {
				s.logger.Warn("error on close reader", "err", errClose)
			}
		}
	}()

	for i := 0; i < file.ChunksCount; i++ {
		objKey := s.makeFileChunkKey(fileID, i)
		reader, _, errObject := s.minio.GetObject(ctx, s.bucket, objKey)
		if errObject != nil {
			errStatus := s.fileRepo.UpdateFileStatus(ctx, fileID, model.FileStatusFailed)
			if errStatus != nil {
				s.logger.Warn("error on update file status", "err", errStatus)
			}
			return model.File{}, fmt.Errorf("get chunk from storage: %w", errObject)
		}
		readers = append(readers, reader)
	}

	var allData []byte
	for _, reader := range readers {
		data, errRead := io.ReadAll(reader)
		if errRead != nil {
			errStatus := s.fileRepo.UpdateFileStatus(ctx, fileID, model.FileStatusFailed)
			if errStatus != nil {
				s.logger.Warn("error on update file status", "err", errStatus)
			}
			return model.File{}, fmt.Errorf("read chunk: %w", errRead)
		}
		allData = append(allData, data...)
	}

	err = s.minio.PutObject(ctx, s.bucket, composedKey, bytes.NewReader(allData), int64(len(allData)))
	if err != nil {
		errStatus := s.fileRepo.UpdateFileStatus(ctx, fileID, model.FileStatusFailed)
		if errStatus != nil {
			s.logger.Warn("error on update file status", "err", errStatus)
		}
		return model.File{}, fmt.Errorf("compose file: %w", err)
	}

	err = s.minio.DeleteObjects(ctx, s.bucket, objectKeys)
	if err != nil {
		s.logger.Warn("error on delete objects", "err", err)
	}

	err = s.fileRepo.UpdateFileStatus(ctx, fileID, model.FileStatusCompleted)
	if err != nil {
		return model.File{}, fmt.Errorf("update file status: %w", err)
	}

	file.Status = model.FileStatusCompleted
	file.UpdatedAt = time.Now()

	return file, nil
}

func (s *FileService) DownloadFile(
	ctx context.Context,
	fileID uuid.UUID,
	userID uuid.UUID,
) (reader io.ReadCloser, mimeType string, size int64, err error) {
	file, err := s.fileRepo.GetFileByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", 0, ErrFileNotFound
		}
		return nil, "", 0, fmt.Errorf("get file: %w", err)
	}

	if file.UserID != userID {
		return nil, "", 0, ErrFileAccessDenied
	}

	if file.Status != model.FileStatusCompleted {
		return nil, "", 0, ErrFileNotCompleted
	}

	objectKey := s.makeFileKey(fileID)
	reader, size, err = s.minio.GetObject(ctx, s.bucket, objectKey)
	if err != nil {
		return nil, "", 0, fmt.Errorf("get object from storage: %w", err)
	}

	return reader, file.MimeType, size, nil
}

func (s *FileService) DeleteFile(
	ctx context.Context,
	fileID uuid.UUID,
	userID uuid.UUID,
) error {
	file, err := s.fileRepo.GetFileByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrFileNotFound
		}
		return fmt.Errorf("get file: %w", err)
	}

	if file.UserID != userID {
		return ErrFileAccessDenied
	}

	objectKey := s.makeFileKey(fileID)
	errDelete := s.minio.DeleteObject(ctx, s.bucket, objectKey)
	if errDelete != nil {
		s.logger.Warn("error on delete file", "err", errDelete)
	}

	for i := 0; i < file.ChunksCount; i++ {
		chunkKey := s.makeFileChunkKey(fileID, i)
		errDelete = s.minio.DeleteObject(ctx, s.bucket, chunkKey)
		if errDelete != nil {
			s.logger.Warn("error on delete file chunk", "err", errDelete)
		}
	}

	err = s.fileRepo.DeleteFile(ctx, fileID)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

func (s *FileService) makeFileKey(fileID uuid.UUID) string {
	return fmt.Sprintf("%s%s", minioFilesPrefix, fileID.String())
}

func (s *FileService) makeFileChunkKey(fileID uuid.UUID, chunkIndex int) string {
	return fmt.Sprintf("%s%s/%d", minioChunksPrefix, fileID.String(), chunkIndex)
}

func (s *FileService) ListFiles(
	ctx context.Context,
	userID uuid.UUID,
	page, pageSize int,
	query *string,
) ([]model.File, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	limit := pageSize
	offset := (page - 1) * pageSize

	files, total, err := s.fileRepo.ListWithCountByUserID(ctx, userID, query, &limit, &offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list files with count: %w", err)
	}

	return files, total, nil
}
