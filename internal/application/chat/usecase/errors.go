package usecase

import "errors"

var (
	ErrCreateParticipant        = errors.New("failed to create participant")
	ErrParticipantAlreadyExists = errors.New("participant already exists")
	ErrParticipantNotFound      = errors.New("participant not found")

	ErrChatRoomNotFound          = errors.New("chat room not found")
	ErrChatRoomCreate            = errors.New("failed to create chat room")
	ErrAlreadyMember             = errors.New("already a member of this room")
	ErrInvitationNotFound        = errors.New("invitation not found")
	ErrInvitationCreate          = errors.New("failed to create invitation")
	ErrInvitationAlreadyExists   = errors.New("join request already pending")
	ErrInvitationAlreadyResolved = errors.New("invitation already resolved")
	ErrNotRoomAdmin              = errors.New("caller is not room owner or admin")
	ErrNotRoomMember             = errors.New("not a member of this room")

	ErrInvalidMessageType = errors.New("invalid message type")
	ErrSendMessage        = errors.New("failed to send message")

	ErrInvalidFileType = errors.New("unsupported file type")
	ErrFileTooLarge    = errors.New("file too large")
	ErrUploadRoomMedia = errors.New("failed to upload room media")
)
