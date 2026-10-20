package files

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrFilenameRequired  = errors.New("filename required")
	ErrMimeTypeRequired  = errors.New("mime type required")
	ErrStorageKeyNeeded  = errors.New("storage key required")
	ErrInvalidFileSize   = errors.New("invalid file size")
	ErrFilenameTooLong   = errors.New("filename too long")
	ErrMimeTypeTooLong   = errors.New("mime type too long")
	ErrStorageKeyTooLong = errors.New("storage key too long")
	ErrFileTooLarge      = errors.New("file too large")
)

type Service struct {
	repo Repository
}

const (
	defaultFilesLimit  = 20
	maxFilesLimit      = 100
	maxFilenameRunes   = 255
	maxMimeTypeRunes   = 255
	maxStorageKeyRunes = 512
	maxFileSizeBytes   = 100 * 1024 * 1024
)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByOwner(ctx context.Context, ownerUserID string, limit, offset int) ([]File, error) {
	normalizedLimit := normalizeFilesLimit(limit)
	normalizedOffset := normalizeFilesOffset(offset)
	files, err := s.repo.ListFilesByOwner(ctx, ownerUserID, normalizedLimit, normalizedOffset)
	if err != nil {
		return nil, err
	}

	payload := make([]File, 0, len(files))
	for _, file := range files {
		payload = append(payload, mapStoredFile(file))
	}
	return payload, nil
}

func (s *Service) Create(ctx context.Context, ownerUserID, filename, mimeType string, sizeBytes int64, storageKey string) (File, error) {
	normalizedFilename := strings.TrimSpace(filename)
	if normalizedFilename == "" {
		return File{}, ErrFilenameRequired
	}
	if utf8.RuneCountInString(normalizedFilename) > maxFilenameRunes {
		return File{}, ErrFilenameTooLong
	}
	normalizedMimeType := strings.TrimSpace(mimeType)
	if normalizedMimeType == "" {
		return File{}, ErrMimeTypeRequired
	}
	if utf8.RuneCountInString(normalizedMimeType) > maxMimeTypeRunes {
		return File{}, ErrMimeTypeTooLong
	}
	if sizeBytes < 0 {
		return File{}, ErrInvalidFileSize
	}
	if sizeBytes > maxFileSizeBytes {
		return File{}, ErrFileTooLarge
	}
	normalizedStorageKey := strings.TrimSpace(storageKey)
	if normalizedStorageKey == "" {
		return File{}, ErrStorageKeyNeeded
	}
	if utf8.RuneCountInString(normalizedStorageKey) > maxStorageKeyRunes {
		return File{}, ErrStorageKeyTooLong
	}

	file, err := s.repo.CreateFile(ctx, ownerUserID, normalizedFilename, normalizedMimeType, sizeBytes, normalizedStorageKey)
	if err != nil {
		return File{}, err
	}
	return mapStoredFile(file), nil
}

func (s *Service) GetByID(ctx context.Context, ownerUserID, fileID string) (File, error) {
	file, err := s.repo.GetFileByID(ctx, ownerUserID, fileID)
	if err != nil {
		if errors.Is(err, ErrFileNotFound) {
			return File{}, ErrFileNotFound
		}
		return File{}, err
	}
	return mapStoredFile(file), nil
}

func mapStoredFile(file StoredFile) File {
	return File{
		ID:          file.ID,
		OwnerUserID: file.OwnerUserID,
		Filename:    file.Filename,
		MimeType:    file.MimeType,
		SizeBytes:   file.SizeBytes,
		StorageKey:  file.StorageKey,
		CreatedAt:   file.CreatedAt,
	}
}

func normalizeFilesLimit(limit int) int {
	if limit <= 0 {
		return defaultFilesLimit
	}
	if limit > maxFilesLimit {
		return maxFilesLimit
	}
	return limit
}

func normalizeFilesOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
