package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysyncdistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumeSelfSenderKeySyncDistributionRecordsReceiptForConsumedCopies(t *testing.T) {
	requesterDeviceID := mustParseSyncTestDeviceID(t, "11111111-1111-4111-8111-111111111111")
	providerDeviceID := mustParseSyncTestDeviceID(t, "22222222-2222-4222-8222-222222222222")
	syncState := &selfsenderkeysync.SelfSenderKeySync{
		ID:                9,
		ParticipantID:     501,
		RequesterDeviceID: requesterDeviceID,
		ProviderDeviceID:  &providerDeviceID,
		Status:            selfsenderkeysync.StatusUploaded,
		RequestedAt:       time.Now().Add(-time.Minute),
		UpdatedAt:         time.Now().Add(-time.Minute),
	}
	copyRepo := &consumeSelfSyncCopyRepoStub{
		pending: map[selfsenderkeysyncdistribution.ID]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution{
			71: {
				ID:                  71,
				SelfSenderKeySyncID: syncState.ID,
				ParticipantID:       syncState.ParticipantID,
				RequesterDeviceID:   requesterDeviceID,
				ProviderDeviceID:    providerDeviceID,
				SenderMemberID:      9101,
				SenderDeviceID:      providerDeviceID,
				SenderKeyVersion:    77,
				Status:              selfsenderkeysyncdistribution.StatusAvailable,
			},
		},
	}
	receiptRepo := &consumeSelfSyncReceiptRepoStub{}
	uc := NewConsumeSelfSenderKeySyncDistributionUseCase(
		&selfSyncParticipantRepoStub{
			participant: &participant.Participant{
				ID:     syncState.ParticipantID,
				Type:   participant.UserType,
				UserID: ptrSharedUserID(701),
			},
		},
		&consumeSelfSyncChatMemberRepoStub{
			byID: map[chatmember.ID]*chatmember.ChatMember{
				9101: {
					ID:            9101,
					RoomID:        61,
					ParticipantID: 888,
					JoinedAt:      time.Now().Add(-time.Hour),
					UpdatedAt:     time.Now().Add(-time.Hour),
				},
			},
			byRoomAndParticipant: map[string]*chatmember.ChatMember{
				"61:501": {
					ID:            9102,
					RoomID:        61,
					ParticipantID: syncState.ParticipantID,
					JoinedAt:      time.Now().Add(-time.Hour),
					UpdatedAt:     time.Now().Add(-time.Hour),
				},
			},
		},
		&consumeSelfSyncRepoStub{current: syncState},
		copyRepo,
		receiptRepo,
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[ConsumeSelfSenderKeySyncDistributionInput]{
		Base: syncMutationBaseContext(601, 701, requesterDeviceID),
		Data: ConsumeSelfSenderKeySyncDistributionInput{
			DistributionID: 71,
			Status:         "consumed",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, selfsenderkeysyncdistribution.StatusConsumed, copyRepo.pending[71].Status)
	require.Len(t, receiptRepo.records, 1)
	assert.Equal(t, chatmember.ID(9101), receiptRepo.records[0].SenderMemberID)
	assert.Equal(t, providerDeviceID, receiptRepo.records[0].SenderDeviceID)
	assert.Equal(t, chatmember.ID(9102), receiptRepo.records[0].ReceiverMemberID)
	assert.Equal(t, requesterDeviceID, receiptRepo.records[0].ReceiverDeviceID)
	assert.Equal(t, int64(77), receiptRepo.records[0].SenderKeyVersion)
	assert.Equal(t, senderkeyreceipt.SourceSelfSync, receiptRepo.records[0].Source)
}

func TestConsumeSelfSenderKeySyncDistributionDoesNotRecordReceiptForFailedCopies(t *testing.T) {
	requesterDeviceID := mustParseSyncTestDeviceID(t, "33333333-3333-4333-8333-333333333333")
	providerDeviceID := mustParseSyncTestDeviceID(t, "44444444-4444-4444-8444-444444444444")
	syncState := &selfsenderkeysync.SelfSenderKeySync{
		ID:                10,
		ParticipantID:     502,
		RequesterDeviceID: requesterDeviceID,
		ProviderDeviceID:  &providerDeviceID,
		Status:            selfsenderkeysync.StatusUploaded,
		RequestedAt:       time.Now().Add(-time.Minute),
		UpdatedAt:         time.Now().Add(-time.Minute),
	}
	copyRepo := &consumeSelfSyncCopyRepoStub{
		pending: map[selfsenderkeysyncdistribution.ID]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution{
			88: {
				ID:                  88,
				SelfSenderKeySyncID: syncState.ID,
				ParticipantID:       syncState.ParticipantID,
				RequesterDeviceID:   requesterDeviceID,
				ProviderDeviceID:    providerDeviceID,
				SenderMemberID:      9201,
				SenderDeviceID:      providerDeviceID,
				SenderKeyVersion:    88,
				Status:              selfsenderkeysyncdistribution.StatusAvailable,
			},
		},
	}
	receiptRepo := &consumeSelfSyncReceiptRepoStub{}
	uc := NewConsumeSelfSenderKeySyncDistributionUseCase(
		&selfSyncParticipantRepoStub{
			participant: &participant.Participant{
				ID:     syncState.ParticipantID,
				Type:   participant.UserType,
				UserID: ptrSharedUserID(702),
			},
		},
		&consumeSelfSyncChatMemberRepoStub{},
		&consumeSelfSyncRepoStub{current: syncState},
		copyRepo,
		receiptRepo,
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[ConsumeSelfSenderKeySyncDistributionInput]{
		Base: syncMutationBaseContext(602, 702, requesterDeviceID),
		Data: ConsumeSelfSenderKeySyncDistributionInput{
			DistributionID: 88,
			Status:         "failed",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, selfsenderkeysyncdistribution.StatusFailed, copyRepo.pending[88].Status)
	assert.Empty(t, receiptRepo.records)
}

type consumeSelfSyncRepoStub struct {
	current *selfsenderkeysync.SelfSenderKeySync
}

func (s *consumeSelfSyncRepoStub) FindByParticipantID(context.Context, participant.ID) (*selfsenderkeysync.SelfSenderKeySync, error) {
	if s.current == nil {
		return nil, selfsenderkeysync.ErrNotFound
	}
	copied := *s.current
	if s.current.ProviderDeviceID != nil {
		providerDeviceID := *s.current.ProviderDeviceID
		copied.ProviderDeviceID = &providerDeviceID
	}
	return &copied, nil
}

func (s *consumeSelfSyncRepoStub) UpsertPending(context.Context, participant.ID, shared.DeviceID) (*selfsenderkeysync.SelfSenderKeySync, error) {
	return nil, nil
}

func (s *consumeSelfSyncRepoStub) ClaimProvider(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID, shared.DeviceID) (bool, error) {
	return false, nil
}

func (s *consumeSelfSyncRepoStub) MarkUploaded(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID, shared.DeviceID) error {
	return nil
}

func (s *consumeSelfSyncRepoStub) MarkCompleted(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID) error {
	return nil
}

func (s *consumeSelfSyncRepoStub) MarkFailed(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID, shared.DeviceID, string, bool) error {
	return nil
}

type consumeSelfSyncCopyRepoStub struct {
	pending map[selfsenderkeysyncdistribution.ID]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution
}

func (s *consumeSelfSyncCopyRepoStub) ReplaceForSync(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID, shared.DeviceID, []*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution) error {
	return nil
}

func (s *consumeSelfSyncCopyRepoStub) FindPendingByRequester(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID) ([]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, error) {
	out := make([]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, 0, len(s.pending))
	for _, row := range s.pending {
		copied := *row
		out = append(out, &copied)
	}
	return out, nil
}

func (s *consumeSelfSyncCopyRepoStub) MarkConsumed(_ context.Context, _ selfsenderkeysync.ID, id selfsenderkeysyncdistribution.ID, _ participant.ID, _ shared.DeviceID) error {
	if row, ok := s.pending[id]; ok {
		now := time.Now()
		row.Status = selfsenderkeysyncdistribution.StatusConsumed
		row.ConsumedAt = &now
		row.FailedAt = nil
	}
	return nil
}

func (s *consumeSelfSyncCopyRepoStub) MarkFailed(_ context.Context, _ selfsenderkeysync.ID, id selfsenderkeysyncdistribution.ID, _ participant.ID, _ shared.DeviceID) error {
	if row, ok := s.pending[id]; ok {
		now := time.Now()
		row.Status = selfsenderkeysyncdistribution.StatusFailed
		row.FailedAt = &now
		row.ConsumedAt = nil
	}
	return nil
}

func (s *consumeSelfSyncCopyRepoStub) HasNonConsumed(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID) (bool, error) {
	return false, nil
}

type consumeSelfSyncReceiptRepoStub struct {
	records []*senderkeyreceipt.SenderKeyReceipt
}

func (s *consumeSelfSyncReceiptRepoStub) FindLatest(context.Context, chatmember.ID, shared.DeviceID) (*senderkeyreceipt.SenderKeyReceipt, error) {
	return nil, senderkeyreceipt.ErrNotFound
}

func (s *consumeSelfSyncReceiptRepoStub) Upsert(_ context.Context, receipt *senderkeyreceipt.SenderKeyReceipt) error {
	copied := *receipt
	s.records = append(s.records, &copied)
	return nil
}

type consumeSelfSyncChatMemberRepoStub struct {
	byID                 map[chatmember.ID]*chatmember.ChatMember
	byRoomAndParticipant map[string]*chatmember.ChatMember
}

func (s *consumeSelfSyncChatMemberRepoStub) FindByID(_ context.Context, id chatmember.ID) (*chatmember.ChatMember, error) {
	if member, ok := s.byID[id]; ok {
		copied := *member
		return &copied, nil
	}
	return nil, chatmember.ErrNotFound
}

func (s *consumeSelfSyncChatMemberRepoStub) FindByRoomAndParticipant(_ context.Context, roomID chatroom.ID, participantID participant.ID) (*chatmember.ChatMember, error) {
	key := stringKeyForRoomParticipant(roomID, participantID)
	if member, ok := s.byRoomAndParticipant[key]; ok {
		copied := *member
		return &copied, nil
	}
	return nil, chatmember.ErrNotFound
}

func (s *consumeSelfSyncChatMemberRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *consumeSelfSyncChatMemberRepoStub) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *consumeSelfSyncChatMemberRepoStub) Add(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *consumeSelfSyncChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *consumeSelfSyncChatMemberRepoStub) SoftDelete(context.Context, chatmember.ID) error {
	return nil
}

func (s *consumeSelfSyncChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

func stringKeyForRoomParticipant(roomID chatroom.ID, participantID participant.ID) string {
	return fmt.Sprintf("%d:%d", roomID, participantID)
}
