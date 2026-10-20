package files

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrFileNotFound = errors.New("file not found")
)

type StoredFile struct {
	ID          string
	OwnerUserID string
	Filename    string
	MimeType    string
	SizeBytes   int64
	StorageKey  string
	CreatedAt   time.Time
}

type Repository interface {
	ListFilesByOwner(ctx context.Context, ownerUserID string, limit, offset int) ([]StoredFile, error)
	CreateFile(ctx context.Context, ownerUserID, filename, mimeType string, sizeBytes int64, storageKey string) (StoredFile, error)
	GetFileByID(ctx context.Context, ownerUserID, fileID string) (StoredFile, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ListFilesByOwner(ctx context.Context, ownerUserID string, limit, offset int) ([]StoredFile, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, owner_user_id, filename, mime_type, size_bytes, storage_key, created_at
		FROM files
		WHERE owner_user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, ownerUserID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]StoredFile, 0)
	for rows.Next() {
		var file StoredFile
		if err := rows.Scan(&file.ID, &file.OwnerUserID, &file.Filename, &file.MimeType, &file.SizeBytes, &file.StorageKey, &file.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return files, nil
}

func (r *PGRepository) CreateFile(ctx context.Context, ownerUserID, filename, mimeType string, sizeBytes int64, storageKey string) (StoredFile, error) {
	var file StoredFile
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO files (owner_user_id, filename, mime_type, size_bytes, storage_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, owner_user_id, filename, mime_type, size_bytes, storage_key, created_at
	`, ownerUserID, filename, mimeType, sizeBytes, storageKey).Scan(
		&file.ID,
		&file.OwnerUserID,
		&file.Filename,
		&file.MimeType,
		&file.SizeBytes,
		&file.StorageKey,
		&file.CreatedAt,
	)
	if err != nil {
		return StoredFile{}, err
	}
	return file, nil
}

func (r *PGRepository) GetFileByID(ctx context.Context, ownerUserID, fileID string) (StoredFile, error) {
	var file StoredFile
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, owner_user_id, filename, mime_type, size_bytes, storage_key, created_at
		FROM files
		WHERE id = $1
		  AND owner_user_id = $2
	`, fileID, ownerUserID).Scan(
		&file.ID,
		&file.OwnerUserID,
		&file.Filename,
		&file.MimeType,
		&file.SizeBytes,
		&file.StorageKey,
		&file.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredFile{}, ErrFileNotFound
		}
		return StoredFile{}, err
	}
	return file, nil
}
