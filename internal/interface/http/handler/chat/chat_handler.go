package chat

import (
	"net/http"
	"strconv"

	appChat "github.com/HiroLiang/goat-server/internal/application/chat"
	"github.com/HiroLiang/goat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

// ChatHandler handles REST endpoints for chat groups and messages.
type ChatHandler struct {
	chatUseCase *appChat.UseCase
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(chatUseCase *appChat.UseCase) *ChatHandler {
	return &ChatHandler{chatUseCase: chatUseCase}
}

// RegisterChatRoutes registers chat-related API routes.
func (h *ChatHandler) RegisterChatRoutes(r *gin.RouterGroup) {
	r.GET("/groups", h.getMyGroups)
	r.GET("/groups/:id/messages", h.getGroupMessages)
	r.POST("/groups", h.createGroup)
}

// @Summary List my chat groups
// @Description Returns all chat groups the current user belongs to, with last message preview and unread count.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetMyGroupsResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/groups [get]
func (h *ChatHandler) getMyGroups(c *gin.Context) {
	output, err := h.chatUseCase.GetMyGroups(c.Request.Context(), adapter.BuildInput(c, appChat.GetMyGroupsInput{}))
	if err != nil {
		HandleError(c, err)
		return
	}

	groups := make([]ChatGroupResponse, 0, len(output.Groups))
	for _, g := range output.Groups {
		item := ChatGroupResponse{
			ID:          g.ID,
			Type:        g.Type,
			Name:        g.Name,
			Description: g.Description,
			AvatarURL:   g.AvatarURL,
			UnreadCount: g.UnreadCount,
			MemberCount: g.MemberCount,
		}
		if g.LastMessage != nil {
			item.LastMessage = &LastMessagePreviewResponse{
				Content:    g.LastMessage.Content,
				SenderName: g.LastMessage.SenderName,
				Timestamp:  g.LastMessage.Timestamp,
			}
		}
		groups = append(groups, item)
	}

	c.JSON(http.StatusOK, GetMyGroupsResponse{Groups: groups})
}

// @Summary Get messages in a chat group
// @Description Returns a page of messages for the given group. Use the nextCursor value as the `before` query param to load older messages.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id   path  int    true  "Chat group ID"
// @Param before query int    false "Load messages older than this message ID (cursor)"
// @Param limit  query int    false "Number of messages to return (default 20, max 50)"
// @Success 200 {object} GetGroupMessagesResponse
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/groups/{id}/messages [get]
func (h *ChatHandler) getGroupMessages(c *gin.Context) {
	groupIDStr := c.Param("id")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAM", "message": "invalid group id"})
		return
	}

	data := appChat.GetGroupMessagesInput{GroupID: groupID}

	if beforeStr := c.Query("before"); beforeStr != "" {
		v, err := strconv.ParseInt(beforeStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAM", "message": "invalid before cursor"})
			return
		}
		data.BeforeID = &v
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		v, err := strconv.ParseUint(limitStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAM", "message": "invalid limit"})
			return
		}
		data.Limit = v
	}

	output, err := h.chatUseCase.GetGroupMessages(c.Request.Context(), adapter.BuildInput(c, data))
	if err != nil {
		HandleError(c, err)
		return
	}

	messages := make([]ChatMessageResponse, 0, len(output.Messages))
	for _, m := range output.Messages {
		messages = append(messages, ChatMessageResponse{
			ID:           m.ID,
			ChatID:       m.ChatID,
			SenderID:     m.SenderID,
			SenderName:   m.SenderName,
			SenderAvatar: m.SenderAvatar,
			Content:      m.Content,
			Type:         string(m.Type),
			ReplyToID:    m.ReplyToID,
			IsEdited:     m.IsEdited,
			IsMe:         m.IsMe,
			Timestamp:    m.Timestamp,
		})
	}

	c.JSON(http.StatusOK, GetGroupMessagesResponse{
		Messages:   messages,
		NextCursor: output.NextCursor,
		HasMore:    output.HasMore,
	})
}

// @Summary Create a chat group
// @Description Creates a new chat group (direct, group, or bot). For direct and bot types, returns the existing group if one already exists with the same participants.
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateGroupRequest true "Create group request"
// @Success 201 {object} CreateGroupResponse "Group created"
// @Success 200 {object} CreateGroupResponse "Group already exists"
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 409 {object} response.ErrorResponse "Conflict"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/chat/groups [post]
func (h *ChatHandler) createGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAM", "message": err.Error()})
		return
	}

	data := appChat.CreateGroupInput{
		Type:        req.Type,
		Name:        req.Name,
		Description: req.Description,
		MemberIDs:   req.MemberIDs,
		MaxMembers:  req.MaxMembers,
	}

	output, err := h.chatUseCase.CreateGroup(c.Request.Context(), adapter.BuildInput(c, data))
	if err != nil {
		HandleError(c, err)
		return
	}

	resp := CreateGroupResponse{
		Group: ChatGroupResponse{
			ID:          output.Group.ID,
			Type:        output.Group.Type,
			Name:        output.Group.Name,
			Description: output.Group.Description,
			AvatarURL:   output.Group.AvatarURL,
			UnreadCount: output.Group.UnreadCount,
			MemberCount: output.Group.MemberCount,
		},
		IsCreated: output.IsCreated,
	}

	if output.IsCreated {
		c.JSON(http.StatusCreated, resp)
	} else {
		c.JSON(http.StatusOK, resp)
	}
}
