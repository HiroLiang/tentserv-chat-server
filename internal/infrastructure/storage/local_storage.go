package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/HiroLiang/goat-server/internal/application/shared/security"
	"github.com/disintegration/imaging"
)

const (
	avatarSize    = 256
	avatarQuality = 85
)

// LocalFileStorage saves files to the local filesystem.
type LocalFileStorage struct {
	basePath string
	hasher   security.Hasher
}

func NewLocalFileStorage(basePath string, hasher security.Hasher) *LocalFileStorage {
	return &LocalFileStorage{basePath: basePath, hasher: hasher}
}

// SaveAvatar center-crops, resizes to 256×256, encodes as JPEG, and saves the avatar.
// Returns the public URL of the saved file.
func (s *LocalFileStorage) SaveAvatar(_ context.Context, userID int64, r io.Reader, oldFilename string) (string, error) {

	// Decode image + auto-orient
	src, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	// Center-crop to square
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	side := w
	if h < w {
		side = h
	}
	cropped := imaging.CropCenter(src, side, side)

	// Resize to avatarSize×avatarSize
	resized := imaging.Resize(cropped, avatarSize, avatarSize, imaging.Lanczos)

	// encode to buffer
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(avatarQuality)); err != nil {
		return "", fmt.Errorf("encode avatar: %w", err)
	}
	data := buf.Bytes()

	// content hash
	fullHash, err := s.hasher.HashBytes(data)
	if err != nil {
		return "", fmt.Errorf("hash avatar: %w", err)
	}
	ver := fullHash[:12]

	// Ensure directory exists
	dir := filepath.Join(s.basePath, "avatars")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create avatar dir: %w", err)
	}

	// Save as JPEG (overwrites existing avatar for the same user)
	filename := fmt.Sprintf("%d_%s.jpg", userID, ver)
	path := filepath.Join(dir, filename)

	// Write the file atomically
	if _, statErr := os.Stat(path); statErr == nil {
		return filename, nil
	} else if os.IsNotExist(statErr) {
		// write atomically: tmp -> rename
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0644); err != nil {
			return "", fmt.Errorf("write temp avatar: %w", err)
		}
		if err := os.Rename(tmp, path); err != nil {
			_ = os.Remove(tmp)
			return "", fmt.Errorf("rename avatar: %w", err)
		}
	} else {
		return "", fmt.Errorf("stat avatar: %w", statErr)
	}

	// Delete the old file
	oldFilename = strings.TrimSpace(oldFilename)
	if oldFilename != "" && oldFilename != filename {
		oldPath := filepath.Join(dir, oldFilename)

		if filepath.Base(oldFilename) == oldFilename {
			_ = os.Remove(oldPath)
		}
	}

	return filename, nil
}
