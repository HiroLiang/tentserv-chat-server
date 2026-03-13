package storage

import (
	"context"
	"io"

	"github.com/HiroLiang/goat-server/internal/application/shared/port"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type LocalFileStorage struct {
}

func NewLocalFileStorage() *LocalFileStorage {
	return &LocalFileStorage{}
}

func (s *LocalFileStorage) Save(ctx context.Context, file shared.File, dest string) (port.SaveResult, error) {
	//TODO implement me
	panic("implement me")
}

func (s *LocalFileStorage) SaveStream(ctx context.Context, reader io.Reader, meta port.FileMeta, dest string) (port.SaveResult, error) {
	//TODO implement me
	panic("implement me")
}

func (s *LocalFileStorage) Delete(ctx context.Context, path string) error {
	//TODO implement me
	panic("implement me")
}

func (s *LocalFileStorage) URL(path string) string {
	//TODO implement me
	panic("implement me")
}

var _ port.FileStorage = (*LocalFileStorage)(nil)
