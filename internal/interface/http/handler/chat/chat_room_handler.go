package chat

import (
	"net/http"
	"strconv"

	"github.com/HiroLiang/goat-server/internal/application/chat/usecase"
	"github.com/HiroLiang/goat-server/internal/domain/chatroom"
	"github.com/HiroLiang/goat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

type ChatRoomHandler struct {
	createChatRoomUseCase      *usecase.CreateChatRoomUseCase
	joinChatRoomUseCase        *usecase.JoinChatRoomUseCase
	approveJoinRequestUseCase  *usecase.ApproveJoinRequestUseCase
	getUserChatRoomsUseCase    *usecase.GetUserChatRoomsUseCase
	getChatRoomDetailUseCase   *usecase.GetChatRoomDetailUseCase
	getChatRoomMessagesUseCase *usecase.GetChatRoomMessagesUseCase
	updateMemberStatusUseCase  *usecase.UpdateMemberStatusUseCase
}

func NewChatRoomHandler(
	createChatRoomUseCase *usecase.CreateChatRoomUseCase,
	joinChatRoomUseCase *usecase.JoinChatRoomUseCase,
	approveJoinRequestUseCase *usecase.ApproveJoinRequestUseCase,
	getUserChatRoomsUseCase *usecase.GetUserChatRoomsUseCase,
	getChatRoomDetailUseCase *usecase.GetChatRoomDetailUseCase,
	getChatRoomMessagesUseCase *usecase.GetChatRoomMessagesUseCase,
	updateMemberStatusUseCase *usecase.UpdateMemberStatusUseCase,
) *ChatRoomHandler {
	return &ChatRoomHandler{
		createChatRoomUseCase:      createChatRoomUseCase,
		joinChatRoomUseCase:        joinChatRoomUseCase,
		approveJoinRequestUseCase:  approveJoinRequestUseCase,
		getUserChatRoomsUseCase:    getUserChatRoomsUseCase,
		getChatRoomDetailUseCase:   getChatRoomDetailUseCase,
		getChatRoomMessagesUseCase: getChatRoomMessagesUseCase,
		updateMemberStatusUseCase:  updateMemberStatusUseCase,
	}
}

func (h *ChatRoomHandler) RegisterChatRoomRoutes(r *gin.RouterGroup) {
	r.GET("/rooms", h.getUserChatRooms)
	r.POST("/room", h.createRoom)
	r.POST("/room/:room_id/join", h.joinRoom)
	r.PATCH("/room/invitations/:invitation_id", h.resolveInvitation)
	r.GET("/room/:room_id", h.getChatRoomDetail)
	r.GET("/room/:room_id/messages", h.getChatRoomMessages)
	r.PATCH("/room/:room_id/member/status", h.updateMemberStatus)
}

// @Summary Create a chat room
// @Description Create a new chat room (direct, group, channel, or bot). The creator becomes the owner member.
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body CreateRoomRequest true "Room creation payload"
// @Success 201 {object} CreateRoomResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/room [post]
func (h *ChatRoomHandler) createRoom(c *gin.Context) {
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	input := adapter.BuildInput(c, usecase.CreateChatRoomInput{
		Name:        req.Name,
		Description: req.Description,
		Type:        chatroom.RoomType(req.Type),
		MaxMembers:  req.MaxMembers,
		AllowAgent:  req.AllowAgent,
	})
	out, err := h.createChatRoomUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, CreateRoomResponse{
		ID:         out.ID,
		Name:       out.Name,
		Type:       out.Type,
		MaxMembers: out.MaxMembers,
		AllowAgent: out.AllowAgent,
		CreatedAt:  out.CreatedAt,
	})
}

// @Summary Join a chat room
// @Description Join an existing chat room. For open rooms the member is created immediately; for invite-only rooms a pending invitation is created.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param room_id path int true "Room ID"
// @Success 200 {object} JoinRoomResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Room Not Found"
// @Failure 409 {object} response.ErrorResponse "Already a member"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/room/{room_id}/join [post]
func (h *ChatRoomHandler) joinRoom(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid room_id"})
		return
	}

	input := adapter.BuildInput(c, usecase.JoinChatRoomInput{
		RoomID: roomID,
	})
	out, err := h.joinChatRoomUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, JoinRoomResponse{
		MemberID:     out.MemberID,
		Role:         out.Role,
		JoinedAt:     out.JoinedAt,
		InvitationID: out.InvitationID,
		Status:       out.Status,
	})
}

// @Summary Get user chat rooms
// @Description Retrieve all chat rooms the authenticated user belongs to, grouped by room type (direct, group, channel, bot).
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetUserChatRoomsResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/rooms [get]
func (h *ChatRoomHandler) getUserChatRooms(c *gin.Context) {
	input := adapter.BuildEmptyInput(c)
	out, err := h.getUserChatRoomsUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	toResponse := func(summaries []usecase.ChatRoomSummary) []ChatRoomSummaryResponse {
		result := make([]ChatRoomSummaryResponse, 0, len(summaries))
		for _, s := range summaries {
			result = append(result, ChatRoomSummaryResponse{
				RoomID:      s.RoomID,
				RoomType:    s.RoomType,
				DisplayName: s.DisplayName,
				AvatarURL:   s.AvatarURL,
				LatestMsg:   s.LatestMsg,
				UnreadCount: s.UnreadCount,
			})
		}
		return result
	}

	c.JSON(http.StatusOK, GetUserChatRoomsResponse{
		Direct:  toResponse(out.Direct),
		Group:   toResponse(out.Group),
		Channel: toResponse(out.Channel),
		Bot:     toResponse(out.Bot),
	})
}

// @Summary Get chat room detail
// @Description Retrieve full detail of a chat room including member list and recent messages.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param room_id path int true "Room ID"
// @Success 200 {object} GetChatRoomDetailResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Room Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/room/{room_id} [get]
func (h *ChatRoomHandler) getChatRoomDetail(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid room_id"})
		return
	}

	input := adapter.BuildInput(c, usecase.GetChatRoomDetailInput{RoomID: roomID})
	out, err := h.getChatRoomDetailUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	members := make([]ChatRoomMemberInfoResponse, 0, len(out.Members))
	for _, m := range out.Members {
		members = append(members, ChatRoomMemberInfoResponse{
			MemberID:      m.MemberID,
			ParticipantID: m.ParticipantID,
			DisplayName:   m.DisplayName,
			AvatarURL:     m.AvatarURL,
			Role:          m.Role,
			LastReadAt:    m.LastReadAt,
			JoinedAt:      m.JoinedAt,
		})
	}

	messages := make([]ChatMessageInfoResponse, 0, len(out.Messages))
	for _, msg := range out.Messages {
		messages = append(messages, ChatMessageInfoResponse{
			MessageID: msg.MessageID,
			SenderID:  msg.SenderID,
			Content:   msg.Content,
			Type:      msg.Type,
			ReplyToID: msg.ReplyToID,
			IsEdited:  msg.IsEdited,
			CreatedAt: msg.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, GetChatRoomDetailResponse{
		RoomID:      out.RoomID,
		RoomType:    out.RoomType,
		Name:        out.Name,
		Description: out.Description,
		AvatarURL:   out.AvatarURL,
		Members:     members,
		Messages:    messages,
	})
}

// @Summary Get chat room messages
// @Description Retrieve paginated messages for a chat room. Use before_id for cursor-based pagination.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param room_id path int true "Room ID"
// @Param before_id query int false "Return messages with ID less than this value (pagination cursor)"
// @Param limit query int false "Maximum number of messages to return"
// @Success 200 {object} GetChatRoomMessagesResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Room Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/room/{room_id}/messages [get]
func (h *ChatRoomHandler) getChatRoomMessages(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid room_id"})
		return
	}

	var req GetChatRoomMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	input := adapter.BuildInput(c, usecase.GetChatRoomMessagesInput{
		RoomID:   roomID,
		BeforeID: req.BeforeID,
		Limit:    req.Limit,
	})
	out, err := h.getChatRoomMessagesUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	messages := make([]ChatMessageInfoResponse, 0, len(out.Messages))
	for _, msg := range out.Messages {
		messages = append(messages, ChatMessageInfoResponse{
			MessageID: msg.MessageID,
			SenderID:  msg.SenderID,
			Content:   msg.Content,
			Type:      msg.Type,
			ReplyToID: msg.ReplyToID,
			IsEdited:  msg.IsEdited,
			CreatedAt: msg.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, GetChatRoomMessagesResponse{
		Messages: messages,
		HasMore:  out.HasMore,
	})
}

// @Summary Update member read status
// @Description Mark the authenticated user's last-read timestamp for the given room, returning all members' statuses.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param room_id path int true "Room ID"
// @Success 200 {object} UpdateMemberStatusResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Room Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/room/{room_id}/member/status [patch]
func (h *ChatRoomHandler) updateMemberStatus(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid room_id"})
		return
	}

	input := adapter.BuildInput(c, usecase.UpdateMemberStatusInput{RoomID: roomID})
	out, err := h.updateMemberStatusUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	statuses := make([]MemberStatusInfoResponse, 0, len(out.Members))
	for _, m := range out.Members {
		statuses = append(statuses, MemberStatusInfoResponse{
			MemberID:   m.MemberID,
			LastReadAt: m.LastReadAt,
		})
	}

	c.JSON(http.StatusOK, UpdateMemberStatusResponse{Members: statuses})
}

// @Summary Resolve a join invitation
// @Description Approve or reject a pending join invitation. Only the room owner/admin can approve.
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param invitation_id path int true "Invitation ID"
// @Param payload body ResolveInvitationRequest true "Approval decision"
// @Success 200 {object} ResolveInvitationResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Invitation Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/room/invitations/{invitation_id} [patch]
func (h *ChatRoomHandler) resolveInvitation(c *gin.Context) {
	invIDStr := c.Param("invitation_id")
	invID, err := strconv.ParseInt(invIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid invitation_id"})
		return
	}

	var req ResolveInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	input := adapter.BuildInput(c, usecase.ApproveJoinRequestInput{
		InvitationID: invID,
		Approve:      req.Approve,
	})
	out, err := h.approveJoinRequestUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ResolveInvitationResponse{
		InvitationID: out.InvitationID,
		Status:       out.Status,
		MemberID:     out.MemberID,
		Role:         out.Role,
		JoinedAt:     out.JoinedAt,
	})
}
