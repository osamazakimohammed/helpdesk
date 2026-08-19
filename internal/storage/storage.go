package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"helpdesk/internal/config"
	"helpdesk/internal/types"
)

type Storage interface {
	Upload(ctx context.Context, filename string, r io.Reader) (storageKey string, checksum string, size int64, err error)
	Download(ctx context.Context, storageKey string) (io.ReadCloser, error)
}

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}
	return &LocalStorage{baseDir: baseDir}, nil
}

func (s *LocalStorage) Upload(ctx context.Context, filename string, r io.Reader) (string, string, int64, error) {
	uuidStr := types.UUIDToString(types.NewUUIDv7())
	safeName := filepath.Base(filename)
	storageKey := fmt.Sprintf("%s_%s", uuidStr, safeName)
	targetPath := filepath.Join(s.baseDir, storageKey)

	f, err := os.Create(targetPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create file on disk: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(f, hasher)

	size, err := io.Copy(multiWriter, r)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to write file content: %w", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	return storageKey, checksum, size, nil
}

func (s *LocalStorage) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	targetPath := filepath.Join(s.baseDir, filepath.Clean(storageKey))
	f, err := os.Open(targetPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	return f, nil
}

func InitStorage(cfg *config.Config) (Storage, error) {
	// For high portability and seamless self-hosting, use local data directory with S3 option
	storageDir := "./data/attachments"
	return NewLocalStorage(storageDir)
}
