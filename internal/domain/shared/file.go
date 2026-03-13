package shared

import (
	"path/filepath"
	"strings"
)

type File struct {
	name     string
	ext      string
	mimeType string
	size     int64
	data     []byte
}

func NewFile(name string, data []byte, mimeType string) (File, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return File{}, ErrInvalidFileName
	}
	return File{
		name:     name,
		ext:      filepath.Ext(name),
		mimeType: mimeType,
		size:     int64(len(data)),
		data:     data,
	}, nil
}
