package mock

import (
	"context"
	"fmt"
	"io"
)

// FileStorage is a no-op mock that returns a fake URL without touching the filesystem.
type FileStorage struct{}

func MockFileStorage() *FileStorage {
	return &FileStorage{}
}

func (s *FileStorage) SaveAvatar(_ context.Context, userID int64, _ io.Reader, _ string) (string, error) {
	return fmt.Sprintf("http://localhost:8080/static/avatars/%d.jpg", userID), nil
}
