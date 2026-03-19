package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type LocalFileStorage struct {
	basePath string
	baseURL  string
}

func NewLocalFileStorage(basePath, baseURL string) *LocalFileStorage {
	return &LocalFileStorage{basePath: basePath, baseURL: baseURL}
}

func (s *LocalFileStorage) Save(ctx context.Context, file shared.File, dest string) (port.SaveResult, error) {
	fullPath := filepath.Join(s.basePath, dest)

	if _, err := os.Stat(fullPath); err == nil {
		return port.SaveResult{}, port.ErrFileAlreadyExists
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return port.SaveResult{}, err
	}

	if err := os.WriteFile(fullPath, file.Data(), 0644); err != nil {
		return port.SaveResult{}, err
	}

	return port.SaveResult{
		Path:     dest,
		URL:      s.URL(dest),
		Filename: filepath.Base(dest),
		Size:     file.Size(),
	}, nil
}

func (s *LocalFileStorage) SaveStream(ctx context.Context, reader io.Reader, meta port.FileMeta, dest string) (port.SaveResult, error) {
	fullPath := filepath.Join(s.basePath, dest)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return port.SaveResult{}, err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return port.SaveResult{}, err
	}
	defer f.Close()

	written, err := io.Copy(f, reader)
	if err != nil {
		return port.SaveResult{}, err
	}

	return port.SaveResult{
		Path:     dest,
		URL:      s.URL(dest),
		Filename: meta.Filename,
		Size:     written,
	}, nil
}

func (s *LocalFileStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)
	err := os.Remove(fullPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *LocalFileStorage) URL(path string) string {
	return s.baseURL + "/" + path
}

var _ port.FileStorage = (*LocalFileStorage)(nil)
