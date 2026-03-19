package port

import (
	"context"
	"io"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type SaveResult struct {
	URL      string
	Path     string
	Filename string
	Size     int64
}

type FileMeta struct {
	Filename string
	MimeType string
	Size     int64
}

type FileStorage interface {
	Save(ctx context.Context, file shared.File, dest string) (SaveResult, error)
	SaveStream(ctx context.Context, reader io.Reader, meta FileMeta, dest string) (SaveResult, error)
	Delete(ctx context.Context, path string) error
	URL(path string) string
}
