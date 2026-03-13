package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"path/filepath"

	appShared "github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/application/shared/port"
	"github.com/HiroLiang/goat-server/internal/application/shared/security"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/disintegration/imaging"
)

const (
	avatarSize    = 256
	avatarQuality = 85
)

type UploadAvatarInput struct {
	Image io.Reader
}

type UploadAvatarOutput struct {
	AvatarPath string
}

type UploadAvatarUseCase struct {
	hasher      security.Hasher
	fileStorage port.FileStorage
	userRepo    user.Repository
}

func NewUploadAvatarUseCase(
	hasher security.Hasher,
	fileStorage port.FileStorage,
	userRepo user.Repository,
) *UploadAvatarUseCase {
	return &UploadAvatarUseCase{
		hasher:      hasher,
		fileStorage: fileStorage,
		userRepo:    userRepo,
	}
}

func (uc *UploadAvatarUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[UploadAvatarInput],
) (*UploadAvatarOutput, error) {

	userData, err := uc.userRepo.FindByID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Decode image and auto orient it
	source, err := imaging.Decode(input.Data.Image, imaging.AutoOrientation(true))
	if err != nil {
		return nil, ErrUploadFile
	}

	// Center-crop to square
	cropped := uc.cropToSquare(source)

	// Resize to avatar size
	resized := imaging.Resize(cropped, avatarSize, avatarSize, imaging.Lanczos)

	// Encode to buffer
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(avatarQuality)); err != nil {
		return nil, ErrUploadFile
	}
	data := buf.Bytes()

	// content hash (to check is the image has changed)
	fullHash, err := uc.hasher.HashBytes(data)
	if err != nil {
		return nil, ErrUploadFile
	}
	ver := fullHash[:12]

	// Ensure directory exists
	dir := filepath.Join("avatars", fmt.Sprintf("%d", userData.ID))

	// Save as JPEG (overwrites existing avatar for the same user)
	fileName := fmt.Sprintf("%d_%s.jpg", userData.ID, ver)
	path := filepath.Join(dir, fileName)
	oldFileName := userData.Avatar
	oldPath := filepath.Join(dir, oldFileName)

	// Prepare file data
	fileData, err := shared.NewFile(fileName, data, "image/jpeg")
	if err != nil {
		return nil, ErrUploadFile
	}

	// Save file
	result, err := uc.fileStorage.Save(ctx, fileData, path)
	if err != nil {
		if errors.Is(err, port.ErrFileAlreadyExists) {

			// Update the current user's avatar name if the file name wrong
			if userData.Avatar != fileName {
				userData.Avatar = fileName
				if err := uc.userRepo.Update(ctx, userData); err != nil {
					return nil, ErrUploadFile
				}
			}
			return &UploadAvatarOutput{AvatarPath: path}, nil
		}
		return nil, ErrUploadFile
	}

	// Update user avatar name
	userData.Avatar = result.Filename
	if err := uc.userRepo.Update(ctx, userData); err != nil {
		return nil, ErrUploadFile
	}

	// Delete old avatar if userData is updated
	if oldFileName != "" && oldFileName != userData.Avatar {
		go func() {
			if err := uc.fileStorage.Delete(context.Background(), oldPath); err != nil {
				logger.Log.Warn(fmt.Sprintf("Delete %s failed: %v", oldPath, err))
			}
		}()
	}

	return &UploadAvatarOutput{AvatarPath: result.Path}, nil
}

func (uc *UploadAvatarUseCase) cropToSquare(source image.Image) *image.NRGBA {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	size := width
	if height < width {
		size = height
	}
	return imaging.CropCenter(source, size, size)
}
