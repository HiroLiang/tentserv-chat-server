package usecase

import (
	"context"
	"errors"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	sharedDomain "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type getInvitationParticipantRepoStub struct {
	findByUserID func(ctx context.Context, userID sharedDomain.UserID) (*participant.Participant, error)
	findByID     func(ctx context.Context, id participant.ID) (*participant.Participant, error)
}

func (s *getInvitationParticipantRepoStub) FindByID(ctx context.Context, id participant.ID) (*participant.Participant, error) {
	if s.findByID != nil {
		return s.findByID(ctx, id)
	}
	return nil, participant.ErrNotFound
}

func (s *getInvitationParticipantRepoStub) FindByUserID(ctx context.Context, userID sharedDomain.UserID) (*participant.Participant, error) {
	if s.findByUserID != nil {
		return s.findByUserID(ctx, userID)
	}
	return nil, participant.ErrNotFound
}

func (s *getInvitationParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *getInvitationParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *getInvitationParticipantRepoStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type getInvitationRepoStub struct {
	findByRoomAndInvitee        func(ctx context.Context, roomID chatroom.ID, inviteeID participant.ID) (*chatinvitation.ChatInvitation, error)
	findPendingByRoomAndInviter func(ctx context.Context, roomID chatroom.ID, inviterID participant.ID) (*chatinvitation.ChatInvitation, error)
}

func (s *getInvitationRepoStub) Create(context.Context, *chatinvitation.ChatInvitation) error {
	return nil
}

func (s *getInvitationRepoStub) FindByID(context.Context, chatinvitation.ID) (*chatinvitation.ChatInvitation, error) {
	return nil, chatinvitation.ErrNotFound
}

func (s *getInvitationRepoStub) FindByRoomAndInvitee(ctx context.Context, roomID chatroom.ID, inviteeID participant.ID) (*chatinvitation.ChatInvitation, error) {
	if s.findByRoomAndInvitee != nil {
		return s.findByRoomAndInvitee(ctx, roomID, inviteeID)
	}
	return nil, chatinvitation.ErrNotFound
}

func (s *getInvitationRepoStub) FindPendingByRoomAndInviter(ctx context.Context, roomID chatroom.ID, inviterID participant.ID) (*chatinvitation.ChatInvitation, error) {
	if s.findPendingByRoomAndInviter != nil {
		return s.findPendingByRoomAndInviter(ctx, roomID, inviterID)
	}
	return nil, chatinvitation.ErrNotFound
}

func (s *getInvitationRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatinvitation.ChatInvitation, error) {
	return nil, nil
}

func (s *getInvitationRepoStub) UpdateStatus(context.Context, chatinvitation.ID, chatinvitation.Status) error {
	return nil
}

type getInvitationUserRepoStub struct {
	findByID func(ctx context.Context, id sharedDomain.UserID) (*user.User, error)
}

func (s *getInvitationUserRepoStub) Create(context.Context, *user.User) (sharedDomain.UserID, error) {
	return 0, nil
}

func (s *getInvitationUserRepoStub) FindByID(ctx context.Context, id sharedDomain.UserID) (*user.User, error) {
	if s.findByID != nil {
		return s.findByID(ctx, id)
	}
	return nil, user.ErrUserNotFound
}

func (s *getInvitationUserRepoStub) FindByAccountID(context.Context, sharedDomain.AccountID) (*[]user.User, error) {
	return nil, nil
}

func (s *getInvitationUserRepoStub) Update(context.Context, *user.User) error {
	return nil
}

func (s *getInvitationUserRepoStub) SearchByName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

func (s *getInvitationUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

func (s *getInvitationUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

func TestGetMyRoomInvitationUseCase_InviteePending(t *testing.T) {
	callerUserID := sharedDomain.UserID(99)
	inviterUserID := sharedDomain.UserID(55)
	avatar := "alice.png"

	uc := NewGetMyRoomInvitationUseCase(
		&getInvitationParticipantRepoStub{
			findByUserID: func(context.Context, sharedDomain.UserID) (*participant.Participant, error) {
				return &participant.Participant{ID: participant.ID(7), UserID: &callerUserID}, nil
			},
			findByID: func(context.Context, participant.ID) (*participant.Participant, error) {
				return &participant.Participant{ID: participant.ID(11), UserID: &inviterUserID}, nil
			},
		},
		&getInvitationRepoStub{
			findByRoomAndInvitee: func(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
				return &chatinvitation.ChatInvitation{
					ID:        chatinvitation.ID(1001),
					RoomID:    chatroom.ID(3),
					InviterID: participant.ID(11),
					InviteeID: participant.ID(7),
					Status:    chatinvitation.Pending,
				}, nil
			},
		},
		&getInvitationUserRepoStub{
			findByID: func(context.Context, sharedDomain.UserID) (*user.User, error) {
				return &user.User{ID: inviterUserID, Name: "Alice", Avatar: avatar}, nil
			},
		},
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetMyRoomInvitationInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: callerUserID},
		},
		Data: GetMyRoomInvitationInput{RoomID: 3},
	})

	require.NoError(t, err)
	assert.True(t, out.Found)
	assert.Equal(t, "invitee", out.Role)
	assert.Equal(t, int64(1001), out.InvitationID)
	assert.Equal(t, "Alice", out.InviterName)
	require.NotNil(t, out.InviterUserID)
	assert.Equal(t, int64(inviterUserID), *out.InviterUserID)
	require.NotNil(t, out.InviterAvatar)
	assert.Equal(t, avatar, *out.InviterAvatar)
}

func TestGetMyRoomInvitationUseCase_InviterPending(t *testing.T) {
	callerUserID := sharedDomain.UserID(42)

	uc := NewGetMyRoomInvitationUseCase(
		&getInvitationParticipantRepoStub{
			findByUserID: func(context.Context, sharedDomain.UserID) (*participant.Participant, error) {
				return &participant.Participant{ID: participant.ID(9), UserID: &callerUserID}, nil
			},
		},
		&getInvitationRepoStub{
			findByRoomAndInvitee: func(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
				return nil, chatinvitation.ErrNotFound
			},
			findPendingByRoomAndInviter: func(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
				return &chatinvitation.ChatInvitation{
					ID:        chatinvitation.ID(2002),
					RoomID:    chatroom.ID(8),
					InviterID: participant.ID(9),
					InviteeID: participant.ID(10),
					Status:    chatinvitation.Pending,
				}, nil
			},
		},
		&getInvitationUserRepoStub{},
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetMyRoomInvitationInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: callerUserID},
		},
		Data: GetMyRoomInvitationInput{RoomID: 8},
	})

	require.NoError(t, err)
	assert.True(t, out.Found)
	assert.Equal(t, "inviter", out.Role)
	assert.Equal(t, int64(2002), out.InvitationID)
	assert.Empty(t, out.InviterName)
	assert.Nil(t, out.InviterUserID)
}

func TestGetMyRoomInvitationUseCase_NoInvitation(t *testing.T) {
	callerUserID := sharedDomain.UserID(7)

	uc := NewGetMyRoomInvitationUseCase(
		&getInvitationParticipantRepoStub{
			findByUserID: func(context.Context, sharedDomain.UserID) (*participant.Participant, error) {
				return &participant.Participant{ID: participant.ID(3), UserID: &callerUserID}, nil
			},
		},
		&getInvitationRepoStub{
			findByRoomAndInvitee: func(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
				return nil, chatinvitation.ErrNotFound
			},
			findPendingByRoomAndInviter: func(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
				return nil, chatinvitation.ErrNotFound
			},
		},
		&getInvitationUserRepoStub{},
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetMyRoomInvitationInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: callerUserID},
		},
		Data: GetMyRoomInvitationInput{RoomID: 11},
	})

	require.NoError(t, err)
	assert.False(t, out.Found)
	assert.Zero(t, out.InvitationID)
}

func TestGetMyRoomInvitationUseCase_QueryErrorWrapped(t *testing.T) {
	callerUserID := sharedDomain.UserID(7)
	dbErr := errors.New("database unavailable")

	uc := NewGetMyRoomInvitationUseCase(
		&getInvitationParticipantRepoStub{
			findByUserID: func(context.Context, sharedDomain.UserID) (*participant.Participant, error) {
				return &participant.Participant{ID: participant.ID(3), UserID: &callerUserID}, nil
			},
		},
		&getInvitationRepoStub{
			findByRoomAndInvitee: func(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
				return nil, dbErr
			},
		},
		&getInvitationUserRepoStub{},
	)

	_, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetMyRoomInvitationInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: callerUserID},
		},
		Data: GetMyRoomInvitationInput{RoomID: 11},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvitationQuery)
	assert.ErrorIs(t, err, dbErr)
}
