package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/HiroLiang/tentserv-chat-server/internal/logger"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/chat/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type SendMessageInput struct {
	RoomID    int64
	Content   string
	Type      string
	ReplyToID *int64
}

type SendMessageOutput struct {
	MessageID int64
	RoomID    int64
	SenderID  int64
	Content   string
	Type      string
	ReplyToID *int64
	CreatedAt time.Time
}

type SendMessageUseCase struct {
	participantRepo participant.Repository
	chatMemberRepo  chatmember.Repository
	chatRoomRepo    chatroom.Repository
	chatMessageRepo chatmessage.Repository
	broadcaster     port.Broadcaster
	friendshipRepo  friendship.Repository
}

func NewSendMessageUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	chatRoomRepo chatroom.Repository,
	chatMessageRepo chatmessage.Repository,
	broadcaster port.Broadcaster,
	friendshipRepo ...friendship.Repository,
) *SendMessageUseCase {
	var fsRepo friendship.Repository
	if len(friendshipRepo) > 0 {
		fsRepo = friendshipRepo[0]
	}
	return &SendMessageUseCase{
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		chatRoomRepo:    chatRoomRepo,
		chatMessageRepo: chatMessageRepo,
		broadcaster:     broadcaster,
		friendshipRepo:  fsRepo,
	}
}

// [EN] Execute: validates caller is an active room member with send permission,
//
//	validates message type (text/image/file), defends against path traversal for file/image content,
//	persists the message, then calls fanOut() asynchronously to broadcast to all room members.
//
// [中] Execute：驗證呼叫者為房間有效成員且有傳訊權限，驗證訊息類型（text/image/file），
//
//	對 file/image 防範路徑遍歷攻擊，持久化訊息後非同步呼叫 fanOut() 廣播給所有成員。
//
// [日] Execute：呼び出し元がアクティブなルームメンバーかつ送信権限を持つことを検証し、
//
//	メッセージタイプ（text/image/file）を検証、ファイル/画像コンテンツのパストラバーサルを防御、
//	メッセージを永続化した後、fanOut() を非同期で呼び出して全メンバーにブロードキャストする。
func (uc *SendMessageUseCase) Execute(
	ctx context.Context,
	input shared.UseCaseInput[SendMessageInput],
) (SendMessageOutput, error) {
	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return SendMessageOutput{}, ErrParticipantNotFound
		}
		return SendMessageOutput{}, err
	}

	roomID := chatroom.ID(input.Data.RoomID)

	room, err := uc.chatRoomRepo.FindByID(ctx, roomID)
	if err != nil || room.IsDeleted {
		return SendMessageOutput{}, ErrChatRoomNotFound
	}

	callerMember, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, callerParticipant.ID)
	if err != nil || callerMember.IsDeleted {
		return SendMessageOutput{}, ErrNotRoomMember
	}

	if !chatmember.CanSendMessage(callerMember.Role) {
		return SendMessageOutput{}, ErrNotAllowed
	}
	if room.Type == chatroom.Direct {
		allMembers, err := uc.chatMemberRepo.FindByRoom(ctx, roomID)
		if err != nil {
			return SendMessageOutput{}, err
		}
		blockedSenders, _, err := blockedMembersForCaller(
			ctx,
			uc.friendshipRepo,
			uc.participantRepo,
			input.Base.Auth.UserID,
			callerParticipant.ID,
			allMembers,
		)
		if err != nil {
			return SendMessageOutput{}, err
		}
		if len(blockedSenders) > 0 {
			return SendMessageOutput{}, ErrUserBlocked
		}
	}

	msgType := chatmessage.MessageType(input.Data.Type)
	switch msgType {
	case chatmessage.Text, chatmessage.Image, chatmessage.File:
		// valid
	default:
		return SendMessageOutput{}, ErrInvalidMessageType
	}

	// Defense-in-depth: for file/image, reject path traversal attempts.
	// Decode percent-encoding first to catch %2e%2e and %2f variants, then
	// clean the path and reject any remaining traversal indicators.
	if msgType == chatmessage.Image || msgType == chatmessage.File {
		decoded, err := url.PathUnescape(input.Data.Content)
		if err != nil {
			return SendMessageOutput{}, ErrInvalidMessageType
		}
		cleaned := path.Clean(decoded)
		if strings.Contains(cleaned, "..") ||
			strings.HasPrefix(cleaned, "/") ||
			strings.Contains(cleaned, "\\") {
			return SendMessageOutput{}, ErrInvalidMessageType
		}
	}

	var replyToID *chatmessage.ID
	if input.Data.ReplyToID != nil {
		id := chatmessage.ID(*input.Data.ReplyToID)
		replyToID = &id
	}

	msg := &chatmessage.ChatMessage{
		RoomID:    roomID,
		SenderID:  callerMember.ID,
		Content:   input.Data.Content,
		Type:      msgType,
		ReplyToID: replyToID,
	}

	if err := uc.chatMessageRepo.Create(ctx, msg); err != nil {
		return SendMessageOutput{}, ErrSendMessage
	}

	out := SendMessageOutput{
		MessageID: int64(msg.ID),
		RoomID:    int64(msg.RoomID),
		SenderID:  int64(msg.SenderID),
		Content:   msg.Content,
		Type:      string(msg.Type),
		CreatedAt: msg.CreatedAt,
	}
	if msg.ReplyToID != nil {
		v := int64(*msg.ReplyToID)
		out.ReplyToID = &v
	}

	go uc.fanOut(out)

	return out, nil
}

type wsMessagePayload struct {
	MessageID int64     `json:"message_id"`
	RoomID    int64     `json:"room_id"`
	SenderID  int64     `json:"sender_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	ReplyToID *int64    `json:"reply_to_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type wsEnvelope struct {
	Type    string           `json:"type"`
	Payload wsMessagePayload `json:"payload"`
}

// [EN] fanOut: runs in a goroutine. Fetches all room members, serializes the chat.message envelope,
//
//	then calls broadcaster.SendToUser() for each member's userID.
//	Online clients receive immediately; offline clients are handled by the delivery queue.
//
// [中] fanOut：在 goroutine 中執行，取得所有房間成員，序列化 chat.message 封包，
//
//	對每個成員的 userID 呼叫 broadcaster.SendToUser()；線上客戶端立即收到，離線由投遞佇列處理。
//
// [日] fanOut：goroutine で実行。全ルームメンバーを取得し、chat.message エンベロープをシリアライズして
//
//	各メンバーの userID に broadcaster.SendToUser() を呼び出す。
//	オンラインクライアントは即座に受信；オフラインは配信キューが処理する。
func (uc *SendMessageUseCase) fanOut(out SendMessageOutput) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error("fanOut panic recovered", zap.Any("recover", r))
		}
	}()

	ctx := context.Background()

	members, err := uc.chatMemberRepo.FindByRoom(ctx, chatroom.ID(out.RoomID))
	if err != nil {
		return
	}

	payload, err := json.Marshal(wsEnvelope{
		Type: "chat.message",
		Payload: wsMessagePayload{
			MessageID: out.MessageID,
			RoomID:    out.RoomID,
			SenderID:  out.SenderID,
			Content:   out.Content,
			Type:      out.Type,
			ReplyToID: out.ReplyToID,
			CreatedAt: out.CreatedAt,
		},
	})
	if err != nil {
		return
	}

	for _, m := range members {
		if m.IsDeleted {
			continue
		}
		p, err := uc.participantRepo.FindByID(ctx, m.ParticipantID)
		if err != nil || p.UserID == nil {
			continue
		}
		userIDStr := strconv.FormatInt(int64(*p.UserID), 10)
		uc.broadcaster.SendToUser(userIDStr, payload)
	}
}
