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

func (f File) Name() string     { return f.name }
func (f File) Ext() string      { return f.ext }
func (f File) MimeType() string { return f.mimeType }
func (f File) Size() int64      { return f.size }
func (f File) Data() []byte     { return f.data }
