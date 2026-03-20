package e2ee

import (
	"net/http"
	"strconv"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

type E2EEHandler struct {
	uploadIdentityKey  *usecase.UploadIdentityKeyUseCase
	uploadSignedPreKey *usecase.UploadSignedPreKeyUseCase
	uploadOTPPreKeys   *usecase.UploadOTPPreKeysUseCase
	countOTPPreKeys    *usecase.CountOTPPreKeysUseCase
	getKeyBundle       *usecase.GetKeyBundleUseCase
	uploadSenderKey    *usecase.UploadSenderKeyUseCase
	getSenderKeys      *usecase.GetSenderKeysUseCase
}

func NewE2EEHandler(
	uploadIdentityKey *usecase.UploadIdentityKeyUseCase,
	uploadSignedPreKey *usecase.UploadSignedPreKeyUseCase,
	uploadOTPPreKeys *usecase.UploadOTPPreKeysUseCase,
	countOTPPreKeys *usecase.CountOTPPreKeysUseCase,
	getKeyBundle *usecase.GetKeyBundleUseCase,
	uploadSenderKey *usecase.UploadSenderKeyUseCase,
	getSenderKeys *usecase.GetSenderKeysUseCase,
) *E2EEHandler {
	return &E2EEHandler{
		uploadIdentityKey:  uploadIdentityKey,
		uploadSignedPreKey: uploadSignedPreKey,
		uploadOTPPreKeys:   uploadOTPPreKeys,
		countOTPPreKeys:    countOTPPreKeys,
		getKeyBundle:       getKeyBundle,
		uploadSenderKey:    uploadSenderKey,
		getSenderKeys:      getSenderKeys,
	}
}

func (h *E2EEHandler) RegisterE2EERoutes(r *gin.RouterGroup) {
	r.POST("/identity-key", h.uploadIdentityKey_)
	r.POST("/signed-prekey", h.uploadSignedPreKey_)
	r.POST("/otp-prekeys", h.uploadOTPPreKeys_)
	r.GET("/otp-prekeys/count", h.countOTPPreKeys_)
	r.GET("/key-bundle/:user_id", h.getKeyBundle_)
	r.POST("/sender-key", h.uploadSenderKey_)
	r.GET("/sender-keys/:room_id", h.getSenderKeys_)
}

// @Summary Upload identity key
// @Description Upload or replace the Curve25519 identity key for a device. Returns the SHA-256 fingerprint.
// @Tags E2EE
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body UploadIdentityKeyRequest true "Identity key payload"
// @Success 200 {object} UploadIdentityKeyResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/identity-key [post]
func (h *E2EEHandler) uploadIdentityKey_(c *gin.Context) {
	var req UploadIdentityKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}
	input := adapter.BuildInput(c, usecase.UploadIdentityKeyInput{
		DeviceID:      req.DeviceID,
		PublicKey:     req.PublicKey,
		SignPublicKey: req.SignPublicKey,
	})
	out, err := h.uploadIdentityKey.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, UploadIdentityKeyResponse{Fingerprint: out.Fingerprint})
}

// @Summary Upload signed pre-key
// @Description Upload or replace the signed pre-key (SPK) for a device, signed by its identity key.
// @Tags E2EE
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body UploadSignedPreKeyRequest true "Signed pre-key payload"
// @Success 204
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/signed-prekey [post]
func (h *E2EEHandler) uploadSignedPreKey_(c *gin.Context) {
	var req UploadSignedPreKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}
	input := adapter.BuildInput(c, usecase.UploadSignedPreKeyInput{
		DeviceID:  req.DeviceID,
		KeyID:     req.KeyID,
		PublicKey: req.PublicKey,
		Signature: req.Signature,
	})
	if _, err := h.uploadSignedPreKey.Execute(c.Request.Context(), input); err != nil {
		HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary Upload one-time pre-keys
// @Description Batch-upload one-time pre-keys (OTP) for a device. Returns the new total count stored on the server.
// @Tags E2EE
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body UploadOTPPreKeysRequest true "OTP pre-keys payload"
// @Success 200 {object} UploadOTPPreKeysResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/otp-prekeys [post]
func (h *E2EEHandler) uploadOTPPreKeys_(c *gin.Context) {
	var req UploadOTPPreKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}
	items := make([]usecase.OTPPreKeyItem, len(req.Keys))
	for i, k := range req.Keys {
		items[i] = usecase.OTPPreKeyItem{KeyID: k.KeyID, PublicKey: k.PublicKey}
	}
	input := adapter.BuildInput(c, usecase.UploadOTPPreKeysInput{
		DeviceID: req.DeviceID,
		Keys:     items,
	})
	out, err := h.uploadOTPPreKeys.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, UploadOTPPreKeysResponse{Count: out.Count})
}

// @Summary Count remaining OTP pre-keys
// @Description Return the number of one-time pre-keys still available on the server for a device.
// @Tags E2EE
// @Produce json
// @Security BearerAuth
// @Param device_id query string true "Device ID"
// @Success 200 {object} CountOTPPreKeysResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/otp-prekeys/count [get]
func (h *E2EEHandler) countOTPPreKeys_(c *gin.Context) {
	deviceID := c.Query("device_id")
	input := adapter.BuildInput(c, usecase.CountOTPPreKeysInput{DeviceID: deviceID})
	out, err := h.countOTPPreKeys.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, CountOTPPreKeysResponse{Count: out.Count})
}

// @Summary Get key bundle for X3DH
// @Description Fetch the identity key, signed pre-key, and one optional OTP pre-key for a target user/device. Used by the initiating party to perform X3DH key agreement.
// @Tags E2EE
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "Target user ID"
// @Param device_id query string true "Target device ID"
// @Success 200 {object} KeyBundleResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Key bundle not found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/key-bundle/{user_id} [get]
func (h *E2EEHandler) getKeyBundle_(c *gin.Context) {
	userID := c.Param("user_id")
	deviceID := c.Query("device_id")

	input := adapter.BuildInput(c, usecase.GetKeyBundleInput{
		TargetUserID: userID,
		DeviceID:     deviceID,
	})
	out, err := h.getKeyBundle.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, KeyBundleResponse{
		IdentityKey:     out.IdentityKey,
		IdentityKeySign: out.IdentityKeySign,
		SignedPreKey:    out.SignedPreKey,
		SPKSignature:    out.SPKSignature,
		SPKKeyID:        out.SPKKeyID,
		OTPPreKey:       out.OTPPreKey,
		OTPPreKeyID:     out.OTPPreKeyID,
	})
}

// @Summary Upload sender key for a group room
// @Description Upload the authenticated member's sender key and SKDM distribution message for a group/channel room.
// @Tags E2EE
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body UploadSenderKeyRequest true "Sender key payload"
// @Success 204
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/sender-key [post]
func (h *E2EEHandler) uploadSenderKey_(c *gin.Context) {
	var req UploadSenderKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}
	input := adapter.BuildInput(c, usecase.UploadSenderKeyInput{
		RoomID:              req.RoomID,
		SenderKeyPublic:     req.SenderKeyPublic,
		DistributionMessage: req.DistributionMessage,
	})
	if _, err := h.uploadSenderKey.Execute(c.Request.Context(), input); err != nil {
		HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary Get sender keys for a group room
// @Description Retrieve all members' sender keys and distribution messages for the given room. Used when joining or re-keying a group session.
// @Tags E2EE
// @Produce json
// @Security BearerAuth
// @Param room_id path int true "Room ID"
// @Success 200 {object} GetSenderKeysResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/sender-keys/{room_id} [get]
func (h *E2EEHandler) getSenderKeys_(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("room_id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	input := adapter.BuildInput(c, usecase.GetSenderKeysInput{RoomID: roomID})
	out, err := h.getSenderKeys.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	items := make([]SenderKeyItemResponse, len(out.Keys))
	for i, k := range out.Keys {
		items[i] = SenderKeyItemResponse{
			ChatMemberID:        k.ChatMemberID,
			SenderKeyPublic:     k.SenderKeyPublic,
			DistributionMessage: k.DistributionMessage,
		}
	}
	c.JSON(http.StatusOK, GetSenderKeysResponse{Keys: items})
}
