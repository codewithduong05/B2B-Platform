package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Storage interface {
	Put(ctx context.Context, key string, data io.Reader) (path string, size int64, checksum string, err error)
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	URL(ctx context.Context, path string, expiresAt time.Time) (string, error)
}

type LocalStorage struct {
	rootDir  string
	baseURL  string
}

func NewLocalStorage(rootDir, baseURL string) (*LocalStorage, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve storage dir: %w", err)
	}
	if err := os.MkdirAll(absRoot, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &LocalStorage{rootDir: absRoot, baseURL: baseURL}, nil
}

func (s *LocalStorage) Put(ctx context.Context, key string, data io.Reader) (string, int64, string, error) {
	cleanKey := filepath.Clean(key)
	if strings.Contains(cleanKey, "..") {
		return "", 0, "", fmt.Errorf("invalid key: path traversal detected")
	}
	path := filepath.Join(s.rootDir, cleanKey)
	
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", 0, "", fmt.Errorf("resolve path: %w", err)
	}
	if !strings.HasPrefix(absPath, s.rootDir) {
		return "", 0, "", fmt.Errorf("invalid key: escapes root directory")
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", 0, "", fmt.Errorf("create dir: %w", err)
	}

	f, err := os.Create(absPath)
	if err != nil {
		return "", 0, "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(f, hasher)
	size, err := io.Copy(writer, data)
	if err != nil {
		return "", 0, "", fmt.Errorf("write file: %w", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	return absPath, size, checksum, nil
}

func (s *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	if !strings.HasPrefix(absPath, s.rootDir) {
		return nil, fmt.Errorf("invalid path: escapes root directory")
	}
	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	if !strings.HasPrefix(absPath, s.rootDir) {
		return fmt.Errorf("invalid path: escapes root directory")
	}
	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func (s *LocalStorage) URL(ctx context.Context, path string, expiresAt time.Time) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if !strings.HasPrefix(absPath, s.rootDir) {
		return "", fmt.Errorf("invalid path: escapes root directory")
	}
	relPath, err := filepath.Rel(s.rootDir, absPath)
	if err != nil {
		relPath = filepath.Base(absPath)
	}
	return fmt.Sprintf("%s/exports/%s", s.baseURL, relPath), nil
}
