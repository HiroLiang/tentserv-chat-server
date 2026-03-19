package participant

import (
	"net/http"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/chat/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

// ParticipantHandler handles participant-related REST endpoints.
type ParticipantHandler struct {
	createUserParticipantUseCase *usecase.CreateUserParticipantUseCase
	getUserParticipantUseCase    *usecase.GetUserParticipantUseCase
}

// NewParticipantHandler creates a new ParticipantHandler with dependencies.
func NewParticipantHandler(
	createUserParticipantUseCase *usecase.CreateUserParticipantUseCase,
	getUserParticipantUseCase *usecase.GetUserParticipantUseCase,
) *ParticipantHandler {
	return &ParticipantHandler{
		createUserParticipantUseCase: createUserParticipantUseCase,
		getUserParticipantUseCase:    getUserParticipantUseCase,
	}
}

// RegisterParticipantRoutes registers participant API routes onto the given router group.
// Auth is enforced at the group level in rest.go.
func (h *ParticipantHandler) RegisterParticipantRoutes(r *gin.RouterGroup) {
	r.POST("/user", h.createUserParticipant)
	r.GET("/me", h.getUserParticipant)
}

// @Summary Register as participant
// @Description Register the authenticated user as a chat participant (idempotent: returns 409 if already registered)
// @Tags Participant
// @Produce json
// @Security BearerAuth
// @Success 201 {object} ParticipantResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 409 {object} response.ErrorResponse "Already registered"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/participant/user [post]
func (h *ParticipantHandler) createUserParticipant(c *gin.Context) {
	input := adapter.BuildEmptyInput(c)
	out, err := h.createUserParticipantUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ParticipantResponse{
		ID:        out.ID,
		UserID:    out.UserID,
		CreatedAt: out.CreatedAt,
	})
}

// @Summary Get my participant record
// @Description Retrieve the authenticated user's chat participant record
// @Tags Participant
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ParticipantResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Not registered yet"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/participant/me [get]
func (h *ParticipantHandler) getUserParticipant(c *gin.Context) {
	input := adapter.BuildEmptyInput(c)
	out, err := h.getUserParticipantUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ParticipantResponse{
		ID:        out.ID,
		UserID:    out.UserID,
		CreatedAt: out.CreatedAt,
	})
}
