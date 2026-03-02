package storage

import (
	"context"
	"io"
)

// FileStorage handles file persistence for the application.
type FileStorage interface {
	SaveAvatar(ctx context.Context, userID int64, r io.Reader, oldFilename string) (url string, err error)
}
