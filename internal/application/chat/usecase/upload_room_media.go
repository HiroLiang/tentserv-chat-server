package usecase

import (
	"context"
	"errors"
	"io"
	"path"
	"path/filepath"
	"strconv"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appPort "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/gofrs/uuid"
)

var allowedMIMETypes = map[string]bool{
	"image/jpeg":               true,
	"image/png":                true,
	"image/webp":               true,
	"image/gif":                true,
	"application/pdf":          true,
	"application/zip":          true,
	"text/plain":               true,
	"application/octet-stream": true,
	"application/msword":       true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
}

type UploadRoomMediaInput struct {
	RoomID   int64
	File     io.Reader
	Filename string
	MimeType string
	Size     int64
}

type UploadRoomMediaOutput struct {
	Path     string
	URL      string
	MimeType string
	Size     int64
}

type UploadRoomMediaUseCase struct {
	participantRepo participant.Repository
	chatMemberRepo  chatmember.Repository
	fileStorage     appPort.FileStorage
}

func NewUploadRoomMediaUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	fileStorage appPort.FileStorage,
) *UploadRoomMediaUseCase {
	return &UploadRoomMediaUseCase{
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		fileStorage:     fileStorage,
	}
}

func (uc *UploadRoomMediaUseCase) Execute(
	ctx context.Context,
	input shared.UseCaseInput[UploadRoomMediaInput],
) (UploadRoomMediaOutput, error) {
	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return UploadRoomMediaOutput{}, ErrParticipantNotFound
		}
		return UploadRoomMediaOutput{}, err
	}

	roomID := chatroom.ID(input.Data.RoomID)

	callerMember, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, callerParticipant.ID)
	if err != nil || callerMember.IsDeleted {
		return UploadRoomMediaOutput{}, ErrNotRoomMember
	}

	if !allowedMIMETypes[input.Data.MimeType] {
		return UploadRoomMediaOutput{}, ErrInvalidFileType
	}

	ext := filepath.Ext(input.Data.Filename)
	id, err := uuid.NewV4()
	if err != nil {
		return UploadRoomMediaOutput{}, ErrUploadRoomMedia
	}
	dest := path.Join("rooms", strconv.FormatInt(input.Data.RoomID, 10), id.String()+ext)

	result, err := uc.fileStorage.SaveStream(ctx, input.Data.File, appPort.FileMeta{
		Filename: input.Data.Filename,
		MimeType: input.Data.MimeType,
		Size:     input.Data.Size,
	}, dest)
	if err != nil {
		return UploadRoomMediaOutput{}, ErrUploadRoomMedia
	}

	return UploadRoomMediaOutput{
		Path:     result.Path,
		URL:      result.URL,
		MimeType: input.Data.MimeType,
		Size:     result.Size,
	}, nil
}
