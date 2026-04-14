package e2ee

import (
	"net/http"
	"strconv"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

type E2EEHandler struct {
	uploadIdentityKey                *usecase.UploadIdentityKeyUseCase
	uploadSignedPreKey               *usecase.UploadSignedPreKeyUseCase
	uploadOTPPreKeys                 *usecase.UploadOTPPreKeysUseCase
	countOTPPreKeys                  *usecase.CountOTPPreKeysUseCase
	getKeyBundle                     *usecase.GetKeyBundleUseCase
	checkKeyStatus                   *usecase.CheckKeyStatusUseCase
	getKeyPolicy                     *usecase.GetKeyPolicyUseCase
	uploadSenderKey                  *usecase.UploadSenderKeyUseCase
	getSenderKeys                    *usecase.GetSenderKeysUseCase
	getSenderKeyDistributionStatus   *usecase.GetSenderKeyDistributionStatusUseCase
	getPendingSenderKeyDistributions *usecase.GetPendingSenderKeyDistributionsUseCase
	consumeSenderKeyDistribution     *usecase.ConsumeSenderKeyDistributionUseCase
	createSenderKeyRequest           *usecase.CreateSenderKeyRequestUseCase
	uploadSelfSenderKeySyncDistributions *usecase.UploadSelfSenderKeySyncDistributionsUseCase
	getPendingSelfSenderKeySyncDistributions *usecase.GetPendingSelfSenderKeySyncDistributionsUseCase
	consumeSelfSenderKeySyncDistribution *usecase.ConsumeSelfSenderKeySyncDistributionUseCase
	getSelfSenderKeySync             *usecase.GetSelfSenderKeySyncUseCase
	selfSenderKeySyncMutation        *usecase.SelfSenderKeySyncMutationUseCase
}

func NewE2EEHandler(
	uploadIdentityKey *usecase.UploadIdentityKeyUseCase,
	uploadSignedPreKey *usecase.UploadSignedPreKeyUseCase,
	uploadOTPPreKeys *usecase.UploadOTPPreKeysUseCase,
	countOTPPreKeys *usecase.CountOTPPreKeysUseCase,
	getKeyBundle *usecase.GetKeyBundleUseCase,
	checkKeyStatus *usecase.CheckKeyStatusUseCase,
	getKeyPolicy *usecase.GetKeyPolicyUseCase,
	uploadSenderKey *usecase.UploadSenderKeyUseCase,
	getSenderKeys *usecase.GetSenderKeysUseCase,
	getSenderKeyDistributionStatus *usecase.GetSenderKeyDistributionStatusUseCase,
	getPendingSenderKeyDistributions *usecase.GetPendingSenderKeyDistributionsUseCase,
	consumeSenderKeyDistribution *usecase.ConsumeSenderKeyDistributionUseCase,
	createSenderKeyRequest *usecase.CreateSenderKeyRequestUseCase,
	uploadSelfSenderKeySyncDistributions *usecase.UploadSelfSenderKeySyncDistributionsUseCase,
	getPendingSelfSenderKeySyncDistributions *usecase.GetPendingSelfSenderKeySyncDistributionsUseCase,
	consumeSelfSenderKeySyncDistribution *usecase.ConsumeSelfSenderKeySyncDistributionUseCase,
	getSelfSenderKeySync *usecase.GetSelfSenderKeySyncUseCase,
	selfSenderKeySyncMutation *usecase.SelfSenderKeySyncMutationUseCase,
) *E2EEHandler {
	return &E2EEHandler{
		uploadIdentityKey:                uploadIdentityKey,
		uploadSignedPreKey:               uploadSignedPreKey,
		uploadOTPPreKeys:                 uploadOTPPreKeys,
		countOTPPreKeys:                  countOTPPreKeys,
		getKeyBundle:                     getKeyBundle,
		checkKeyStatus:                   checkKeyStatus,
		getKeyPolicy:                     getKeyPolicy,
		uploadSenderKey:                  uploadSenderKey,
		getSenderKeys:                    getSenderKeys,
		getSenderKeyDistributionStatus:   getSenderKeyDistributionStatus,
		getPendingSenderKeyDistributions: getPendingSenderKeyDistributions,
		consumeSenderKeyDistribution:     consumeSenderKeyDistribution,
		createSenderKeyRequest:           createSenderKeyRequest,
		uploadSelfSenderKeySyncDistributions: uploadSelfSenderKeySyncDistributions,
		getPendingSelfSenderKeySyncDistributions: getPendingSelfSenderKeySyncDistributions,
		consumeSelfSenderKeySyncDistribution: consumeSelfSenderKeySyncDistribution,
		getSelfSenderKeySync:             getSelfSenderKeySync,
		selfSenderKeySyncMutation:        selfSenderKeySyncMutation,
	}
}

func (h *E2EEHandler) RegisterE2EERoutes(r *gin.RouterGroup) {
	r.POST("/identity-key", h.uploadIdentityKey_)
	r.POST("/signed-prekey", h.uploadSignedPreKey_)
	r.POST("/otp-prekeys", h.uploadOTPPreKeys_)
	r.GET("/otp-prekeys/count", h.countOTPPreKeys_)
	r.GET("/key-bundle/:user_id", h.getKeyBundle_)
	r.GET("/key-status/:user_id", h.checkKeyStatus_)
	r.GET("/key-policy", h.getKeyPolicy_)
	r.POST("/sender-key", h.uploadSenderKey_)
	r.GET("/sender-keys/:room_id", h.getSenderKeys_)
	r.GET("/sender-key-distributions/:room_id", h.getSenderKeyDistributionStatus_)
	r.GET("/sender-key-distributions/:room_id/pending", h.getPendingSenderKeyDistributions_)
	r.POST("/sender-key-distributions/:distribution_id/consume", h.consumeSenderKeyDistribution_)
	r.POST("/sender-key-request", h.createSenderKeyRequest_)
	r.POST("/self-sender-key-sync/distributions/bulk", h.bulkSelfSenderKeySyncDistributions_)
	r.GET("/self-sender-key-sync/distributions/pending", h.getPendingSelfSenderKeySyncDistributions_)
	r.POST("/self-sender-key-sync/distributions/:distribution_id/consume", h.consumeSelfSenderKeySyncDistribution_)
	r.GET("/self-sender-key-sync", h.getSelfSenderKeySync_)
	r.POST("/self-sender-key-sync/accept", h.acceptSelfSenderKeySync_)
	r.POST("/self-sender-key-sync/uploaded", h.markSelfSenderKeySyncUploaded_)
	r.POST("/self-sender-key-sync/complete", h.completeSelfSenderKeySync_)
	r.POST("/self-sender-key-sync/fail", h.failSelfSenderKeySync_)
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
// @Description Fetch the identity key, signed pre-key, and one optional OTP pre-key for a target user/device.
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

// @Summary Check key status (non-consuming)
// @Description Check whether the identity key and signed pre-key exist for a user/device. Does NOT consume any OTP pre-key.
// @Tags E2EE
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "Target user ID"
// @Param device_id query string false "Target device ID"
// @Success 200 {object} CheckKeyStatusResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/key-status/{user_id} [get]
func (h *E2EEHandler) checkKeyStatus_(c *gin.Context) {
	userID := c.Param("user_id")
	deviceID := c.Query("device_id")

	input := adapter.BuildInput(c, usecase.CheckKeyStatusInput{
		TargetUserID: userID,
		DeviceID:     deviceID,
	})
	out, err := h.checkKeyStatus.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, CheckKeyStatusResponse{
		IdentityKeyExists:  out.IdentityKeyExists,
		SignedPreKeyExists: out.SignedPreKeyExists,
		DeviceID:           out.DeviceID,
		IdentityKey:        out.IdentityKey,
		IdentityKeySign:    out.IdentityKeySign,
		SignedPreKey:       out.SignedPreKey,
		SPKSignature:       out.SPKSignature,
		SPKKeyID:           out.SPKKeyID,
		OTPPreKeyCount:     out.OTPPreKeyCount,
	})
}

// @Summary Get E2EE key bootstrap policy
// @Description Return server-side OTP target count and replenish threshold for login-time key bootstrap.
// @Tags E2EE
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetKeyPolicyResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/key-policy [get]
func (h *E2EEHandler) getKeyPolicy_(c *gin.Context) {
	out, err := h.getKeyPolicy.Execute(c.Request.Context(), adapter.BuildInput(c, usecase.GetKeyPolicyInput{}))
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, GetKeyPolicyResponse{
		OTPPreKeyTargetCount:        out.OTPPreKeyTargetCount,
		OTPPreKeyReplenishThreshold: out.OTPPreKeyReplenishThreshold,
	})
}

// @Summary Upload sender key for a room
// @Description Upload the authenticated member's sender key and SKDM distribution message.
//
//	Works for both direct and group rooms — every member has their own Sender Key.
//
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
		SenderMemberID:      req.SenderMemberID,
		ReceiverUserID:      req.ReceiverUserID,
		ReceiverDeviceID:    req.ReceiverDeviceID,
		SenderKeyVersion:    req.SenderKeyVersion,
		DistributionMessage: req.DistributionMessage,
	})
	if _, err := h.uploadSenderKey.Execute(c.Request.Context(), input); err != nil {
		HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary Get sender keys for a room
// @Description Retrieve all members' sender keys for the given room (direct or group).
//
//	Also records that the caller has fetched each key (distribution ACK).
//
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
			ChatMemberID:     k.ChatMemberID,
			ProviderDeviceID: k.ProviderDeviceID,
			SenderKeyVersion: k.SenderKeyVersion,
		}
	}
	c.JSON(http.StatusOK, GetSenderKeysResponse{Keys: items})
}

// @Summary Get sender key distribution status for a room
// @Description Returns whether my sender key exists plus room sender-key reconciliation lists:
//
//	requestable_member_ids and pending_from_members for peers whose latest key I still need,
//	available_from_member_ids for peers who already uploaded a consumable distribution to me,
//	available_to_member_ids for members who already have my latest available distribution,
//	and pending_receivers for members who still need a fresh upload from me.
//
// @Tags E2EE
// @Produce json
// @Security BearerAuth
// @Param room_id path int true "Room ID"
// @Success 200 {object} GetSenderKeyDistributionStatusResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/sender-key-distributions/{room_id} [get]
func (h *E2EEHandler) getSenderKeyDistributionStatus_(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("room_id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	input := adapter.BuildInput(c, usecase.GetSenderKeyDistributionStatusInput{RoomID: roomID})
	out, err := h.getSenderKeyDistributionStatus.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetSenderKeyDistributionStatusResponse{
		OwnMemberSenderKeyExists: out.OwnMemberSenderKeyExists,
		RequestableSources:       normalizeRouteRefs(out.RequestableSources),
		AvailableFromSources:     normalizeRouteRefs(out.AvailableFromSources),
		AvailableToTargets:       normalizeRouteRefs(out.AvailableToTargets),
		PendingReceivers:         normalizeRouteRefs(out.PendingReceivers),
		PendingFromSources:       normalizeRouteRefs(out.PendingFromSources),
	})
}

// @Summary Get pending sender key distributions for a room
// @Description Returns all available sender key distributions addressed to the authenticated member in the room.
// @Tags E2EE
// @Produce json
// @Security BearerAuth
// @Param room_id path int true "Room ID"
// @Success 200 {object} GetPendingSenderKeyDistributionsResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/sender-key-distributions/{room_id}/pending [get]
func (h *E2EEHandler) getPendingSenderKeyDistributions_(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("room_id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	input := adapter.BuildInput(c, usecase.GetPendingSenderKeyDistributionsInput{RoomID: roomID})
	out, err := h.getPendingSenderKeyDistributions.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	items := make([]PendingSenderKeyDistributionItemResponse, 0, len(out.Distributions))
	for _, dist := range out.Distributions {
		items = append(items, PendingSenderKeyDistributionItemResponse{
			DistributionID:      dist.DistributionID,
			SenderMemberID:      dist.SenderMemberID,
			SenderDeviceID:      dist.SenderDeviceID,
			ReceiverMemberID:    dist.ReceiverMemberID,
			ReceiverDeviceID:    dist.ReceiverDeviceID,
			SenderKeyVersion:    dist.SenderKeyVersion,
			DistributionMessage: dist.DistributionMessage,
		})
	}

	c.JSON(http.StatusOK, GetPendingSenderKeyDistributionsResponse{Distributions: items})
}

// @Summary Consume sender key distribution
// @Description Marks a sender key distribution as consumed or failed after the client processes it.
// @Tags E2EE
// @Accept json
// @Security BearerAuth
// @Param distribution_id path int true "Distribution ID"
// @Param payload body ConsumeSenderKeyDistributionRequest true "Consume payload"
// @Success 204
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/sender-key-distributions/{distribution_id}/consume [post]
func (h *E2EEHandler) consumeSenderKeyDistribution_(c *gin.Context) {
	distributionID, err := strconv.ParseInt(c.Param("distribution_id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	var req ConsumeSenderKeyDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}

	input := adapter.BuildInput(c, usecase.ConsumeSenderKeyDistributionInput{
		DistributionID: distributionID,
		Status:         req.Status,
	})
	if _, err := h.consumeSenderKeyDistribution.Execute(c.Request.Context(), input); err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Request sender key from another member
// @Description Creates a request for provider_member_id to upload their sender key for the given room.
//
//	Also pushes a real-time e2ee.sender_key_needed notification to the provider if they are online.
//
// @Tags E2EE
// @Accept json
// @Security BearerAuth
// @Param payload body CreateSenderKeyRequestRequest true "Request payload"
// @Success 204
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/e2ee/sender-key-request [post]
func (h *E2EEHandler) createSenderKeyRequest_(c *gin.Context) {
	var req CreateSenderKeyRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}
	input := adapter.BuildInput(c, usecase.CreateSenderKeyRequestInput{
		RoomID:            req.RoomID,
		ProviderUserID:    req.ProviderUserID,
		ProviderDeviceID:  req.ProviderDeviceID,
		SenderMemberID:    req.SenderMemberID,
		RequesterDeviceID: req.RequesterDeviceID,
	})
	if _, err := h.createSenderKeyRequest.Execute(c.Request.Context(), input); err != nil {
		HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func normalizeRouteRefs(refs []usecase.SenderKeyRouteRef) []SenderKeyRouteRefResponse {
	if refs == nil {
		return []SenderKeyRouteRefResponse{}
	}
	items := make([]SenderKeyRouteRefResponse, len(refs))
	for i, ref := range refs {
		items[i] = SenderKeyRouteRefResponse{
			UserID:   ref.UserID,
			MemberID: ref.MemberID,
			DeviceID: ref.DeviceID,
		}
	}
	return items
}

func (h *E2EEHandler) getSelfSenderKeySync_(c *gin.Context) {
	input := adapter.BuildInput(c, usecase.GetSelfSenderKeySyncInput{})
	out, err := h.getSelfSenderKeySync.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSelfSenderKeySyncResponse(out))
}

func (h *E2EEHandler) bulkSelfSenderKeySyncDistributions_(c *gin.Context) {
	var req BulkSelfSenderKeySyncDistributionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}
	items := make([]usecase.SelfSenderKeySyncDistributionUploadItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, usecase.SelfSenderKeySyncDistributionUploadItem{
			SenderMemberID:      item.SenderMemberID,
			SenderDeviceID:      item.SenderDeviceID,
			SenderKeyVersion:    item.SenderKeyVersion,
			DistributionMessage: item.DistributionMessage,
		})
	}
	out, err := h.uploadSelfSenderKeySyncDistributions.Execute(c.Request.Context(), adapter.BuildInput(c, usecase.BulkSelfSenderKeySyncDistributionsInput{
		Items: items,
	}))
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, BulkSelfSenderKeySyncDistributionsResponse{Count: out.Count})
}

func (h *E2EEHandler) getPendingSelfSenderKeySyncDistributions_(c *gin.Context) {
	out, err := h.getPendingSelfSenderKeySyncDistributions.Execute(c.Request.Context(), adapter.BuildInput(c, usecase.GetPendingSelfSenderKeySyncDistributionsInput{}))
	if err != nil {
		HandleError(c, err)
		return
	}
	items := make([]PendingSelfSenderKeySyncDistributionItemResponse, 0, len(out.Distributions))
	for _, item := range out.Distributions {
		items = append(items, PendingSelfSenderKeySyncDistributionItemResponse{
			DistributionID:      item.DistributionID,
			SenderMemberID:      item.SenderMemberID,
			SenderDeviceID:      item.SenderDeviceID,
			SenderKeyVersion:    item.SenderKeyVersion,
			DistributionMessage: item.DistributionMessage,
		})
	}
	c.JSON(http.StatusOK, GetPendingSelfSenderKeySyncDistributionsResponse{Distributions: items})
}

func (h *E2EEHandler) consumeSelfSenderKeySyncDistribution_(c *gin.Context) {
	distributionID, err := strconv.ParseInt(c.Param("distribution_id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	var req ConsumeSelfSenderKeySyncDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}
	if _, err := h.consumeSelfSenderKeySyncDistribution.Execute(c.Request.Context(), adapter.BuildInput(c, usecase.ConsumeSelfSenderKeySyncDistributionInput{
		DistributionID: distributionID,
		Status:         req.Status,
	})); err != nil {
		HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *E2EEHandler) acceptSelfSenderKeySync_(c *gin.Context) {
	input := adapter.BuildInput(c, usecase.AcceptSelfSenderKeySyncInput{})
	out, err := h.selfSenderKeySyncMutation.Accept(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSelfSenderKeySyncResponse(out))
}

func (h *E2EEHandler) markSelfSenderKeySyncUploaded_(c *gin.Context) {
	input := adapter.BuildInput(c, usecase.MarkSelfSenderKeySyncUploadedInput{})
	out, err := h.selfSenderKeySyncMutation.MarkUploaded(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSelfSenderKeySyncResponse(out))
}

func (h *E2EEHandler) completeSelfSenderKeySync_(c *gin.Context) {
	input := adapter.BuildInput(c, usecase.CompleteSelfSenderKeySyncInput{})
	out, err := h.selfSenderKeySyncMutation.Complete(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSelfSenderKeySyncResponse(out))
}

func (h *E2EEHandler) failSelfSenderKeySync_(c *gin.Context) {
	var req FailSelfSenderKeySyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}
	input := adapter.BuildInput(c, usecase.FailSelfSenderKeySyncInput{
		LastError: req.LastError,
		Retryable: req.Retryable,
	})
	out, err := h.selfSenderKeySyncMutation.Fail(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSelfSenderKeySyncResponse(out))
}

func toSelfSenderKeySyncResponse(snapshot *usecase.SelfSenderKeySyncSnapshot) GetSelfSenderKeySyncResponse {
	if snapshot == nil {
		return GetSelfSenderKeySyncResponse{Exists: false, Status: "idle"}
	}
	return GetSelfSenderKeySyncResponse{
		Exists:                 snapshot.Exists,
		Status:                 snapshot.Status,
		RequesterDevice:        toSelfSenderKeyDevice(snapshot.RequesterDevice),
		ProviderDevice:         toSelfSenderKeyDevice(snapshot.ProviderDevice),
		RequesterCurrentDevice: snapshot.RequesterCurrentDevice,
		ProviderCurrentDevice:  snapshot.ProviderCurrentDevice,
		LastError:              snapshot.LastError,
		RequestedAtMS:          snapshot.RequestedAtMS,
		ProviderClaimedAtMS:    snapshot.ProviderClaimedAtMS,
		UploadedAtMS:           snapshot.UploadedAtMS,
		CompletedAtMS:          snapshot.CompletedAtMS,
		FailedAtMS:             snapshot.FailedAtMS,
	}
}

func toSelfSenderKeyDevice(device *usecase.SelfSenderKeySyncDeviceSnapshot) *SelfSenderKeyDevice {
	if device == nil {
		return nil
	}
	return &SelfSenderKeyDevice{
		DeviceID:      device.DeviceID,
		DeviceName:    device.DeviceName,
		Platform:      device.Platform,
		LastIP:        device.LastIP,
		BindingStatus: device.BindingStatus,
	}
}
