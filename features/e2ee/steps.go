package e2ee

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	accountfeatures "github.com/HiroLiang/tentserv-chat-server/features/account"
	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	domaindevice "github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysyncdistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/cucumber/godog"
)

type steps struct {
	*bddsupport.APITestContext
	deps                       *Deps
	accountBDD                 *accountfeatures.Deps
	authHeader                 string
	start                      time.Time
	lastDistributionID         int64
	lastSelfSyncDistributionID int64
}

type keyStatusResponse struct {
	IdentityKeyExists  bool   `json:"identity_key_exists"`
	SignedPreKeyExists bool   `json:"signed_pre_key_exists"`
	DeviceID           string `json:"device_id"`
	IdentityKey        string `json:"identity_key"`
	IdentityKeySign    string `json:"identity_key_sign"`
	SignedPreKey       string `json:"signed_pre_key"`
	SPKSignature       string `json:"spk_signature"`
	SPKKeyID           uint32 `json:"spk_key_id"`
	OTPPreKeyCount     int    `json:"otp_prekey_count"`
}

type keyBundleResponse struct {
	IdentityKey     string  `json:"identity_key"`
	IdentityKeySign string  `json:"identity_key_sign"`
	SignedPreKey    string  `json:"signed_pre_key"`
	SPKSignature    string  `json:"spk_signature"`
	SPKKeyID        uint32  `json:"spk_key_id"`
	OTPPreKey       *string `json:"otp_pre_key"`
	OTPPreKeyID     *uint32 `json:"otp_pre_key_id"`
}

type senderKeyDistributionStatusResponse struct {
	OwnMemberSenderKeyExists bool                     `json:"own_member_sender_key_exists"`
	RequestableSources       []senderKeyDeviceRefStep `json:"requestable_sources"`
	AvailableFromSources     []senderKeyDeviceRefStep `json:"available_from_sources"`
	AvailableToTargets       []senderKeyDeviceRefStep `json:"available_to_targets"`
	PendingReceivers         []senderKeyDeviceRefStep `json:"pending_receivers"`
	PendingFromSources       []senderKeyDeviceRefStep `json:"pending_from_sources"`
}

type senderKeyDeviceRefStep struct {
	UserID   int64  `json:"user_id"`
	MemberID int64  `json:"member_id"`
	DeviceID string `json:"device_id"`
}

type pendingSenderKeyDistributionsResponse struct {
	Distributions []struct {
		DistributionID      int64  `json:"distribution_id"`
		SenderMemberID      int64  `json:"sender_member_id"`
		SenderDeviceID      string `json:"sender_device_id"`
		ReceiverMemberID    int64  `json:"receiver_member_id"`
		ReceiverDeviceID    string `json:"receiver_device_id"`
		SenderKeyVersion    int64  `json:"sender_key_version"`
		DistributionMessage string `json:"distribution_message"`
	} `json:"distributions"`
}

type selfSenderKeySyncResponse struct {
	Exists                 bool   `json:"exists"`
	Status                 string `json:"status"`
	RequesterCurrentDevice bool   `json:"requester_current_device"`
	ProviderCurrentDevice  bool   `json:"provider_current_device"`
	LastError              string `json:"last_error"`
}

type pendingSelfSenderKeySyncDistributionsResponse struct {
	Distributions []struct {
		DistributionID      int64  `json:"distribution_id"`
		SenderMemberID      int64  `json:"sender_member_id"`
		SenderDeviceID      string `json:"sender_device_id"`
		SenderKeyVersion    int64  `json:"sender_key_version"`
		DistributionMessage string `json:"distribution_message"`
	} `json:"distributions"`
}

func RegisterSteps(ctx *godog.ScenarioContext, apiCtx *bddsupport.APITestContext, deps *Deps, accountBDD *accountfeatures.Deps) {
	s := &steps{APITestContext: apiCtx, deps: deps, accountBDD: accountBDD}

	ctx.Step(`^E2EE key bootstrap state is clean$`, s.e2eeKeyBootstrapStateIsClean)
	ctx.Step(`^I request the E2EE key policy using the login token$`, s.iRequestTheE2EEKeyPolicyUsingTheLoginToken)
	ctx.Step(`^the E2EE key policy should be target (\d+) and threshold (\d+)$`, s.theE2EEKeyPolicyShouldBeTargetAndThreshold)
	ctx.Step(`^I request E2EE key status for the logged in user and device "([^"]*)"$`, s.iRequestE2EEKeyStatusForTheLoggedInUserAndDevice)
	ctx.Step(`^E2EE key status should be empty for device "([^"]*)"$`, s.e2eeKeyStatusShouldBeEmptyForDevice)
	ctx.Step(`^I upload E2EE identity key "([^"]*)" for device "([^"]*)"$`, s.iUploadE2EEIdentityKeyForDevice)
	ctx.Step(`^I upload E2EE signed pre-key "([^"]*)" with key id (\d+) for device "([^"]*)"$`, s.iUploadE2EESignedPreKeyWithKeyIDForDevice)
	ctx.Step(`^I upload (\d+) E2EE one-time pre-keys starting at key id (\d+) for device "([^"]*)"$`, s.iUploadE2EEOneTimePreKeysStartingAtKeyIDForDevice)
	ctx.Step(`^E2EE key status should expose identity "([^"]*)", signed pre-key "([^"]*)", key id (\d+), and (\d+) OTP keys for device "([^"]*)"$`, s.e2eeKeyStatusShouldExposeIdentitySignedPreKeyKeyIDAndOTPKeysForDevice)
	ctx.Step(`^E2EE key status should expose identity "([^"]*)", no signed pre-key, and (\d+) OTP keys for device "([^"]*)"$`, s.e2eeKeyStatusShouldExposeIdentityWithoutSignedPreKeyForDevice)
	ctx.Step(`^the E2EE identity key response fingerprint should equal SHA-256 of key "([^"]*)"$`, s.theE2EEIdentityKeyResponseFingerprintShouldEqualSHA256OfKey)
	ctx.Step(`^the E2EE key status check should not consume OTP keys$`, s.theE2EEKeyStatusCheckShouldNotConsumeOTPKeys)
	ctx.Step(`^I upload an invalid E2EE identity key for device "([^"]*)"$`, s.iUploadAnInvalidE2EEIdentityKeyForDevice)
	ctx.Step(`^I upload an invalid E2EE signed pre-key for device "([^"]*)"$`, s.iUploadAnInvalidE2EESignedPreKeyForDevice)
	ctx.Step(`^I upload invalid E2EE one-time pre-keys for device "([^"]*)"$`, s.iUploadInvalidE2EEOneTimePreKeysForDevice)
	ctx.Step(`^E2EE key repositories should remain empty$`, s.e2eeKeyRepositoriesShouldRemainEmpty)
	ctx.Step(`^I call the E2EE endpoint "([^"]*)" with "([^"]*)" authentication$`, s.iCallTheE2EEEndpointWithAuthentication)
	ctx.Step(`^I request E2EE key bundle for the logged in user and device "([^"]*)"$`, s.iRequestE2EEKeyBundleForTheLoggedInUserAndDevice)
	ctx.Step(`^E2EE key bundle should include OTP key id (\d+) and server should have (\d+) OTP keys for device "([^"]*)"$`, s.e2eeKeyBundleShouldIncludeOTPKeyIDAndServerShouldHaveOTPKeysForDevice)
	ctx.Step(`^E2EE key bundle should omit OTP and server should have (\d+) OTP keys for device "([^"]*)"$`, s.e2eeKeyBundleShouldOmitOTPAndServerShouldHaveOTPKeysForDevice)
	ctx.Step(`^E2EE OTP replenish event should be queued for device "([^"]*)"$`, s.e2eeOTPReplenishEventShouldBeQueuedForDevice)
	ctx.Step(`^no E2EE OTP replenish event should be queued$`, s.noE2EEOTPReplenishEventShouldBeQueued)

	// Sender key request steps
	ctx.Step(`^E2EE keys are bootstrapped for the logged in user$`, s.e2eeKeysAreBootstrappedForTheLoggedInUser)
	ctx.Step(`^a room member setup exists with room id (\d+), caller member id (\d+), and provider member id (\d+) in the same room with no existing sender key$`, s.roomMemberSetupSameRoomNoKey)
	ctx.Step(`^a room member setup exists with room id (\d+), caller member id (\d+), and provider member id (\d+) in the same room with an available latest distribution for the caller$`, s.roomMemberSetupSameRoomWithAvailableDistribution)
	ctx.Step(`^a room member setup exists with room id (\d+), caller member id (\d+), and provider member id (\d+) in the same room with an available latest distribution and a pending sender key request for the caller$`, s.roomMemberSetupSameRoomWithAvailableDistributionAndPendingRequest)
	ctx.Step(`^a room member setup exists with room id (\d+), caller member id (\d+), and provider member id (\d+) where provider is in a different room$`, s.roomMemberSetupDifferentRoom)
	ctx.Step(`^a room member setup exists with room id (\d+), provider member id (\d+) in that room, and the caller has no room membership$`, s.roomMemberSetupCallerNotInRoom)
	ctx.Step(`^caller member (\d+) and provider member (\d+) are blocked from each other$`, s.callerAndProviderAreBlocked)
	ctx.Step(`^I create a sender key request for room (\d+) and provider member (\d+)$`, s.iCreateASenderKeyRequestForRoomAndProviderMember)
	ctx.Step(`^a sender key request row should exist from member (\d+) to provider (\d+)$`, s.aSenderKeyRequestRowShouldExist)
	ctx.Step(`^no sender key request row should exist from member (\d+) to provider (\d+)$`, s.noSenderKeyRequestRowShouldExist)
	ctx.Step(`^pending sender key request count from member (\d+) to provider (\d+) should be (\d+)$`, s.pendingSenderKeyRequestCountFromMemberToProviderShouldBe)

	// Sender key distribution steps
	ctx.Step(`^a sender key provider setup exists with room id (\d+), provider member id (\d+), and receiver member id (\d+) in the same room$`, s.senderKeyProviderSetup)
	ctx.Step(`^a sender key receiver setup exists with room id (\d+), sender member id (\d+), and receiver member id (\d+) with available distribution version (\d+)$`, s.senderKeyReceiverSetupWithAvailableDistribution)
	ctx.Step(`^I provide a sender key distribution for room (\d+) to receiver member (\d+) with sender key version (\d+)$`, s.iProvideASenderKeyDistribution)
	ctx.Step(`^I request sender key distribution status for room (\d+)$`, s.iRequestSenderKeyDistributionStatus)
	ctx.Step(`^sender key distribution status should show own key exists as (true|false)$`, s.senderKeyDistributionStatusShouldShowOwnKeyExists)
	ctx.Step(`^sender key distribution status should list available sender member (\d+)$`, s.senderKeyDistributionStatusShouldListAvailableSenderMember)
	ctx.Step(`^sender key distribution status should list available sender member (\d+) device "([^"]*)"$`, s.senderKeyDistributionStatusShouldListAvailableSenderMemberDevice)
	ctx.Step(`^sender key distribution status should list available receiver member (\d+)$`, s.senderKeyDistributionStatusShouldListAvailableReceiverMember)
	ctx.Step(`^sender key distribution status should list available receiver member (\d+) device "([^"]*)"$`, s.senderKeyDistributionStatusShouldListAvailableReceiverMemberDevice)
	ctx.Step(`^sender key distribution status should list pending receiver member (\d+)$`, s.senderKeyDistributionStatusShouldListPendingReceiverMember)
	ctx.Step(`^sender key distribution status should list pending receiver member (\d+) device "([^"]*)"$`, s.senderKeyDistributionStatusShouldListPendingReceiverMemberDevice)
	ctx.Step(`^I list pending sender key distributions for room (\d+)$`, s.iListPendingSenderKeyDistributions)
	ctx.Step(`^pending sender key distributions should include sender member (\d+), receiver member (\d+), and version (\d+)$`, s.pendingSenderKeyDistributionsShouldInclude)
	ctx.Step(`^pending sender key distributions should include sender member (\d+) device "([^"]*)", receiver member (\d+) device "([^"]*)", and version (\d+)$`, s.pendingSenderKeyDistributionsShouldIncludeDevices)
	ctx.Step(`^I mark the first pending sender key distribution as "([^"]*)"$`, s.iMarkTheFirstPendingSenderKeyDistributionAs)
	ctx.Step(`^the first pending sender key distribution should now be "([^"]*)"$`, s.theFirstPendingSenderKeyDistributionShouldNowBe)
	ctx.Step(`^an e2ee\.sender_key_needed event should have been broadcast for provider member (\d+)$`, s.senderKeyNeededEventShouldHaveBeenBroadcast)
	ctx.Step(`^a self sender key sync exists for the logged in user with participant id (\d+), requester device "([^"]*)", and status "([^"]*)"$`, s.aSelfSenderKeySyncExistsForTheLoggedInUserWithoutProvider)
	ctx.Step(`^a self sender key sync exists for the logged in user with participant id (\d+), requester device "([^"]*)", provider device "([^"]*)", and status "([^"]*)"$`, s.aSelfSenderKeySyncExistsForTheLoggedInUserWithProvider)
	ctx.Step(`^the self sender key sync requester device "([^"]*)" is bound as "([^"]*)"$`, s.theSelfSenderKeySyncRequesterDeviceIsBoundAs)
	ctx.Step(`^the self sender key sync provider device "([^"]*)" is bound as "([^"]*)"$`, s.theSelfSenderKeySyncProviderDeviceIsBoundAs)
	ctx.Step(`^the current logged in user is available to sender key use cases$`, s.theCurrentLoggedInUserIsAvailableToSenderKeyUseCases)
	ctx.Step(`^a self sender key sync distribution exists for the logged in user with participant id (\d+), room id (\d+), sender member id (\d+), requester device "([^"]*)", provider device "([^"]*)", and sender key version (\d+)$`, s.aSelfSenderKeySyncDistributionExistsForTheLoggedInUser)
	ctx.Step(`^a self sender key sync distribution exists for the current logged in participant with room id (\d+), sender member id (\d+), requester device "([^"]*)", provider device "([^"]*)", and sender key version (\d+)$`, s.aSelfSenderKeySyncDistributionExistsForTheCurrentLoggedInParticipant)
	ctx.Step(`^self sender key sync snapshot lookups will fail after mutation$`, s.selfSenderKeySyncSnapshotLookupsWillFailAfterMutation)
	ctx.Step(`^I list pending self sender key sync distributions$`, s.iListPendingSelfSenderKeySyncDistributions)
	ctx.Step(`^pending self sender key sync distributions should include sender member (\d+) device "([^"]*)" and version (\d+)$`, s.pendingSelfSenderKeySyncDistributionsShouldInclude)
	ctx.Step(`^I mark the first pending self sender key sync distribution as "([^"]*)"$`, s.iMarkTheFirstPendingSelfSenderKeySyncDistributionAs)
	ctx.Step(`^the first pending self sender key sync distribution should now be "([^"]*)"$`, s.theFirstPendingSelfSenderKeySyncDistributionShouldNowBe)
	ctx.Step(`^a self sender key receipt should exist from sender member (\d+) device "([^"]*)" to requester device "([^"]*)" with version (\d+) and source "([^"]*)"$`, s.aSelfSenderKeyReceiptShouldExistFromSenderMemberDeviceToRequesterDeviceWithVersionAndSource)
	ctx.Step(`^I accept the self sender key sync$`, s.iAcceptTheSelfSenderKeySync)
	ctx.Step(`^I accept the self sender key sync as device "([^"]*)"$`, s.iAcceptTheSelfSenderKeySyncAsDevice)
	ctx.Step(`^I mark the self sender key sync uploaded$`, s.iMarkTheSelfSenderKeySyncUploaded)
	ctx.Step(`^I mark the self sender key sync uploaded as device "([^"]*)"$`, s.iMarkTheSelfSenderKeySyncUploadedAsDevice)
	ctx.Step(`^I complete the self sender key sync$`, s.iCompleteTheSelfSenderKeySync)
	ctx.Step(`^I fail the self sender key sync with last error "([^"]*)" and retryable (true|false)$`, s.iFailTheSelfSenderKeySyncWithLastErrorAndRetryable)
	ctx.Step(`^the self sender key sync response should show status "([^"]*)"$`, s.theSelfSenderKeySyncResponseShouldShowStatus)
	ctx.Step(`^the self sender key sync requester device "([^"]*)" should now be bound as "([^"]*)"$`, s.theSelfSenderKeySyncRequesterDeviceShouldNowBeBoundAs)
}

func (s *steps) e2eeKeyBootstrapStateIsClean() error {
	s.start = time.Now()
	s.authHeader = ""
	s.lastDistributionID = 0
	s.lastSelfSyncDistributionID = 0
	s.deps.Reset()

	identity, signed, otp := s.deps.RepositoryCounts()
	fmt.Println("Given: E2EE key bootstrap state is clean")
	fmt.Printf("Input: identity_keys=%d signed_prekeys=%d otp_prekeys=%d\n", identity, signed, otp)
	fmt.Println("Action: reset in-memory E2EE key repositories")
	fmt.Printf("Output: identity_keys=%d signed_prekeys=%d otp_prekeys=%d\n", identity, signed, otp)
	fmt.Println("Mutation: E2EE repositories reset")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iRequestTheE2EEKeyPolicyUsingTheLoginToken() error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: an authenticated E2EE session exists")
	fmt.Printf("Input: auth_header_present=%t\n", authHeader != "")
	fmt.Println("Action: GET /api/e2ee/key-policy")

	if err := s.DoRequestWithHeaders(http.MethodGet, "/api/e2ee/key-policy", s.authHeaders(authHeader, "")); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theE2EEKeyPolicyShouldBeTargetAndThreshold(target, threshold int) error {
	start := time.Now()
	fmt.Println("Given: E2EE key policy response is available")
	fmt.Printf("Input: expected_target=%d expected_threshold=%d\n", target, threshold)
	fmt.Println("Action: decode key policy response")

	var policy struct {
		OTPPreKeyTargetCount        int `json:"otp_prekey_target_count"`
		OTPPreKeyReplenishThreshold int `json:"otp_prekey_replenish_threshold"`
	}
	if err := json.Unmarshal(s.ResponseBody, &policy); err != nil {
		return err
	}

	matches := policy.OTPPreKeyTargetCount == target && policy.OTPPreKeyReplenishThreshold == threshold
	fmt.Printf("Output: actual_target=%d actual_threshold=%d match=%t\n",
		policy.OTPPreKeyTargetCount, policy.OTPPreKeyReplenishThreshold, matches)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !matches {
		return fmt.Errorf("expected E2EE policy target=%d threshold=%d, got target=%d threshold=%d",
			target, threshold, policy.OTPPreKeyTargetCount, policy.OTPPreKeyReplenishThreshold)
	}
	return nil
}

func (s *steps) iRequestE2EEKeyStatusForTheLoggedInUserAndDevice(deviceID string) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	userID := s.accountBDD.LastSessionUserID()
	if userID == 0 {
		return fmt.Errorf("no logged-in user id available")
	}

	fmt.Println("Given: key status should be read without consuming OTP pre-keys")
	fmt.Printf("Input: user_id=%d device_id=%s token_present=%t\n", userID, deviceID, authHeader != "")
	fmt.Printf("Action: GET /api/e2ee/key-status/%d\n", userID)

	path := fmt.Sprintf("/api/e2ee/key-status/%d?device_id=%s", userID, deviceID)
	if err := s.DoRequestWithHeaders(http.MethodGet, path, s.authHeaders(authHeader, deviceID)); err != nil {
		return err
	}

	statusSummary := "not_decoded"
	if s.Response.StatusCode == http.StatusOK {
		if status, decodeErr := decodeKeyStatus(s.ResponseBody); decodeErr == nil {
			statusSummary = fmt.Sprintf(
				"identity_exists=%t spk_exists=%t public_key_present=%t spk_key_id=%d otp_count=%d",
				status.IdentityKeyExists,
				status.SignedPreKeyExists,
				status.IdentityKey != "" || status.SignedPreKey != "",
				status.SPKKeyID,
				status.OTPPreKeyCount,
			)
		}
	}
	fmt.Printf("Output: status=%d %s\n", s.Response.StatusCode, statusSummary)
	fmt.Printf("Mutation: otp_consume_calls=%d\n", s.deps.OTPConsumeCalls())
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) e2eeKeyStatusShouldBeEmptyForDevice(deviceID string) error {
	start := time.Now()
	fmt.Println("Given: no remote E2EE key material has been uploaded")
	fmt.Printf("Input: expected_device_id=%s\n", deviceID)
	fmt.Println("Action: decode key status response")

	status, err := decodeKeyStatus(s.ResponseBody)
	if err != nil {
		return err
	}
	empty := !status.IdentityKeyExists && !status.SignedPreKeyExists && status.OTPPreKeyCount == 0
	fmt.Printf("Output: identity_exists=%t spk_exists=%t otp_count=%d empty=%t\n",
		status.IdentityKeyExists, status.SignedPreKeyExists, status.OTPPreKeyCount, empty)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !empty {
		return fmt.Errorf("expected empty E2EE key status, got %+v", status)
	}
	return nil
}

func (s *steps) iUploadE2EEIdentityKeyForDevice(keyName, deviceID string) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: local identity public keys are ready for upload")
	fmt.Printf("Input: key_name=%s device_id=%s public_key_present=true\n", keyName, deviceID)
	fmt.Println("Action: POST /api/e2ee/identity-key")

	payload := map[string]any{
		"device_id":       deviceID,
		"public_key":      namedKeyMaterial(keyName, 32),
		"sign_public_key": namedSignKeyMaterial(keyName, 32),
	}
	if err := s.doJSONRequestWithHeaders(http.MethodPost, "/api/e2ee/identity-key", payload, s.authHeaders(authHeader, deviceID)); err != nil {
		return err
	}

	identity, signed, otp := s.deps.RepositoryCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: identity_keys=%d signed_prekeys=%d otp_prekeys=%d\n", identity, signed, otp)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iUploadE2EESignedPreKeyWithKeyIDForDevice(keyName string, keyID int, deviceID string) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: local signed pre-key public material is ready for upload")
	fmt.Printf("Input: key_name=%s key_id=%d device_id=%s public_key_present=true signature_present=true\n", keyName, keyID, deviceID)
	fmt.Println("Action: POST /api/e2ee/signed-prekey")

	payload := map[string]any{
		"device_id":  deviceID,
		"key_id":     keyID,
		"public_key": namedSignedPreKeyMaterial(keyName, 32),
		"signature":  namedSignatureMaterial(keyName, 64),
	}
	if err := s.doJSONRequestWithHeaders(http.MethodPost, "/api/e2ee/signed-prekey", payload, s.authHeaders(authHeader, deviceID)); err != nil {
		return err
	}

	identity, signed, otp := s.deps.RepositoryCounts()
	fmt.Printf("Output: status=%d body_len=%d\n", s.Response.StatusCode, len(s.ResponseBody))
	fmt.Printf("Mutation: identity_keys=%d signed_prekeys=%d otp_prekeys=%d\n", identity, signed, otp)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iUploadE2EEOneTimePreKeysStartingAtKeyIDForDevice(count, startID int, deviceID string) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: local OTP public keys are ready for upload")
	fmt.Printf("Input: count=%d start_key_id=%d device_id=%s raw_keys_redacted=true\n", count, startID, deviceID)
	fmt.Println("Action: POST /api/e2ee/otp-prekeys")

	keys := make([]map[string]any, count)
	for i := range count {
		keys[i] = map[string]any{
			"key_id":     startID + i,
			"public_key": otpKeyMaterial(startID+i, 32),
		}
	}
	payload := map[string]any{"device_id": deviceID, "keys": keys}
	if err := s.doJSONRequestWithHeaders(http.MethodPost, "/api/e2ee/otp-prekeys", payload, s.authHeaders(authHeader, deviceID)); err != nil {
		return err
	}

	identity, signed, otp := s.deps.RepositoryCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: identity_keys=%d signed_prekeys=%d otp_prekeys=%d\n", identity, signed, otp)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) e2eeKeyStatusShouldExposeIdentitySignedPreKeyKeyIDAndOTPKeysForDevice(identityName, spkName string, keyID, otpCount int, deviceID string) error {
	start := time.Now()
	fmt.Println("Given: E2EE key status response should expose public-only material")
	fmt.Printf("Input: identity_name=%s spk_name=%s key_id=%d expected_otp_count=%d device_id=%s\n",
		identityName, spkName, keyID, otpCount, deviceID)
	fmt.Println("Action: decode key status response")

	status, err := decodeKeyStatus(s.ResponseBody)
	if err != nil {
		return err
	}
	matches := status.IdentityKeyExists &&
		status.SignedPreKeyExists &&
		status.DeviceID == deviceID &&
		status.IdentityKey == namedKeyMaterial(identityName, 32) &&
		status.IdentityKeySign == namedSignKeyMaterial(identityName, 32) &&
		status.SignedPreKey == namedSignedPreKeyMaterial(spkName, 32) &&
		status.SPKSignature == namedSignatureMaterial(spkName, 64) &&
		status.SPKKeyID == uint32(keyID) &&
		status.OTPPreKeyCount == otpCount

	fmt.Printf("Output: identity_exists=%t spk_exists=%t device_id=%s key_id=%d otp_count=%d public_match=%t\n",
		status.IdentityKeyExists, status.SignedPreKeyExists, status.DeviceID, status.SPKKeyID, status.OTPPreKeyCount, matches)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !matches {
		return fmt.Errorf("unexpected E2EE key status: %+v", status)
	}
	return nil
}

func (s *steps) e2eeKeyStatusShouldExposeIdentityWithoutSignedPreKeyForDevice(identityName string, otpCount int, deviceID string) error {
	start := time.Now()
	fmt.Println("Given: E2EE key status response should expose identity without signed pre-key")
	fmt.Printf("Input: identity_name=%s expected_otp_count=%d device_id=%s\n", identityName, otpCount, deviceID)
	fmt.Println("Action: decode key status response")

	status, err := decodeKeyStatus(s.ResponseBody)
	if err != nil {
		return err
	}
	matches := status.IdentityKeyExists &&
		!status.SignedPreKeyExists &&
		status.DeviceID == deviceID &&
		status.IdentityKey == namedKeyMaterial(identityName, 32) &&
		status.IdentityKeySign == namedSignKeyMaterial(identityName, 32) &&
		status.SignedPreKey == "" &&
		status.SPKSignature == "" &&
		status.SPKKeyID == 0 &&
		status.OTPPreKeyCount == otpCount

	fmt.Printf("Output: identity_exists=%t spk_exists=%t device_id=%s otp_count=%d partial_match=%t\n",
		status.IdentityKeyExists, status.SignedPreKeyExists, status.DeviceID, status.OTPPreKeyCount, matches)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !matches {
		return fmt.Errorf("unexpected identity-only E2EE key status: %+v", status)
	}
	return nil
}

func (s *steps) theE2EEIdentityKeyResponseFingerprintShouldEqualSHA256OfKey(keyName string) error {
	start := time.Now()
	fmt.Println("Given: identity key upload returns a deterministic fingerprint")
	fmt.Printf("Input: key_name=%s\n", keyName)
	fmt.Println("Action: decode upload response and compare fingerprint")

	var body struct {
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}

	keyBytes, err := base64.StdEncoding.DecodeString(namedKeyMaterial(keyName, 32))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(keyBytes)
	expected := hex.EncodeToString(sum[:])
	isLowerHex64, err := regexp.MatchString("^[0-9a-f]{64}$", body.Fingerprint)
	if err != nil {
		return err
	}
	matches := isLowerHex64 && body.Fingerprint == expected

	fmt.Printf("Output: fingerprint_len=%d lowercase_hex_64=%t match=%t\n",
		len(body.Fingerprint), isLowerHex64, matches)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !matches {
		return fmt.Errorf("expected fingerprint %s, got %s", expected, body.Fingerprint)
	}
	return nil
}

func (s *steps) theE2EEKeyStatusCheckShouldNotConsumeOTPKeys() error {
	start := time.Now()
	fmt.Println("Given: key status must be non-consuming")
	fmt.Println("Input: expected_consume_calls=0")
	fmt.Println("Action: inspect OTP fake repository")

	consumeCalls := s.deps.OTPConsumeCalls()
	fmt.Printf("Output: otp_consume_calls=%d non_consuming=%t\n", consumeCalls, consumeCalls == 0)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if consumeCalls != 0 {
		return fmt.Errorf("expected key status to avoid consuming OTP keys, got consume_calls=%d", consumeCalls)
	}
	return nil
}

func (s *steps) iUploadAnInvalidE2EEIdentityKeyForDevice(deviceID string) error {
	return s.uploadInvalidPayload("/api/e2ee/identity-key", map[string]any{
		"device_id":       deviceID,
		"public_key":      "not-base64",
		"sign_public_key": namedSignKeyMaterial("alpha", 32),
	})
}

func (s *steps) iUploadAnInvalidE2EESignedPreKeyForDevice(deviceID string) error {
	return s.uploadInvalidPayload("/api/e2ee/signed-prekey", map[string]any{
		"device_id":  deviceID,
		"key_id":     1,
		"public_key": shortKeyMaterial(31),
		"signature":  namedSignatureMaterial("alpha", 64),
	})
}

func (s *steps) iUploadInvalidE2EEOneTimePreKeysForDevice(deviceID string) error {
	return s.uploadInvalidPayload("/api/e2ee/otp-prekeys", map[string]any{
		"device_id": deviceID,
		"keys": []map[string]any{{
			"key_id":     1,
			"public_key": shortKeyMaterial(31),
		}},
	})
}

func (s *steps) uploadInvalidPayload(path string, payload map[string]any) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: invalid E2EE public key material is submitted")
	fmt.Printf("Input: path=%s raw_key_bytes_redacted=true\n", path)
	fmt.Printf("Action: POST %s\n", path)

	if err := s.doJSONRequestWithHeaders(http.MethodPost, path, payload, s.authHeaders(authHeader, "")); err != nil {
		return err
	}

	identity, signed, otp := s.deps.RepositoryCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: identity_keys=%d signed_prekeys=%d otp_prekeys=%d\n", identity, signed, otp)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) e2eeKeyRepositoriesShouldRemainEmpty() error {
	start := time.Now()
	fmt.Println("Given: invalid E2EE key upload should not write key rows")
	fmt.Println("Input: expected_identity=0 expected_signed=0 expected_otp=0")
	fmt.Println("Action: inspect in-memory key repositories")

	identity, signed, otp := s.deps.RepositoryCounts()
	empty := identity == 0 && signed == 0 && otp == 0
	fmt.Printf("Output: identity_keys=%d signed_prekeys=%d otp_prekeys=%d empty=%t\n", identity, signed, otp, empty)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !empty {
		return fmt.Errorf("expected E2EE repositories to remain empty, got identity=%d signed=%d otp=%d", identity, signed, otp)
	}
	return nil
}

func (s *steps) iCallTheE2EEEndpointWithAuthentication(endpoint, authMode string) error {
	s.start = time.Now()
	deviceID := s.accountBDD.LastSessionDeviceID().String()
	if deviceID == "" {
		return fmt.Errorf("no logged-in device available for endpoint call")
	}

	headers := map[string]string{
		"X-Device-ID": deviceID,
	}
	switch authMode {
	case "no token":
	case "revoked token":
		authHeader, err := s.authorizationHeader()
		if err != nil {
			return err
		}
		if err := s.accountBDD.RevokeLastAccessToken(); err != nil {
			return err
		}
		headers["Authorization"] = authHeader
	default:
		return fmt.Errorf("unsupported auth mode %q", authMode)
	}

	method, path, payload, err := s.e2eeEndpointRequest(endpoint, deviceID)
	if err != nil {
		return err
	}

	fmt.Println("Given: a protected E2EE endpoint is invoked with invalid authentication")
	fmt.Printf("Input: endpoint=%s auth_mode=%s method=%s device_id=%s payload_present=%t\n",
		endpoint, authMode, method, deviceID, payload != nil)
	fmt.Printf("Action: %s %s\n", method, path)

	if payload != nil {
		if err := s.doJSONRequestWithHeaders(method, path, payload, headers); err != nil {
			return err
		}
	} else if err := s.DoRequestWithHeaders(method, path, headers); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iRequestE2EEKeyBundleForTheLoggedInUserAndDevice(deviceID string) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	userID := s.accountBDD.LastSessionUserID()
	if userID == 0 {
		return fmt.Errorf("no logged-in user id available")
	}

	fmt.Println("Given: a requester needs a target user's public X3DH bundle")
	fmt.Printf("Input: user_id=%d device_id=%s token_present=%t\n", userID, deviceID, authHeader != "")
	fmt.Printf("Action: GET /api/e2ee/key-bundle/%d\n", userID)

	path := fmt.Sprintf("/api/e2ee/key-bundle/%d?device_id=%s", userID, deviceID)
	if err := s.DoRequestWithHeaders(http.MethodGet, path, s.authHeaders(authHeader, deviceID)); err != nil {
		return err
	}

	summary := "not_decoded"
	if s.Response.StatusCode == http.StatusOK {
		if bundle, decodeErr := decodeKeyBundle(s.ResponseBody); decodeErr == nil {
			otpKeyID := uint32(0)
			if bundle.OTPPreKeyID != nil {
				otpKeyID = *bundle.OTPPreKeyID
			}
			summary = fmt.Sprintf(
				"identity_present=%t spk_present=%t otp_present=%t otp_key_id=%d",
				bundle.IdentityKey != "",
				bundle.SignedPreKey != "",
				bundle.OTPPreKey != nil,
				otpKeyID,
			)
		}
	}
	fmt.Printf("Output: status=%d %s\n", s.Response.StatusCode, summary)
	fmt.Printf("Mutation: otp_consume_calls=%d\n", s.deps.OTPConsumeCalls())
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) e2eeKeyBundleShouldIncludeOTPKeyIDAndServerShouldHaveOTPKeysForDevice(keyID, expectedCount int, deviceIDText string) error {
	start := time.Now()
	fmt.Println("Given: key-bundle response may include a consumed OTP pre-key")
	fmt.Printf("Input: expected_otp_key_id=%d expected_remaining_count=%d device_id=%s\n", keyID, expectedCount, deviceIDText)
	fmt.Println("Action: decode key bundle response and inspect OTP repository")

	bundle, err := decodeKeyBundle(s.ResponseBody)
	if err != nil {
		return err
	}
	deviceID, err := shared.ParseDeviceID(deviceIDText)
	if err != nil {
		return err
	}
	userID := s.accountBDD.LastSessionUserID()
	remaining := s.deps.OTPCount(userID, deviceID)
	hasExpectedOTP := bundle.OTPPreKey != nil && bundle.OTPPreKeyID != nil && *bundle.OTPPreKeyID == uint32(keyID)
	matches := bundle.IdentityKey != "" && bundle.SignedPreKey != "" && hasExpectedOTP && remaining == expectedCount

	fmt.Printf("Output: identity_present=%t spk_present=%t otp_present=%t otp_key_id_match=%t remaining_count=%d match=%t\n",
		bundle.IdentityKey != "", bundle.SignedPreKey != "", bundle.OTPPreKey != nil, hasExpectedOTP, remaining, matches)
	fmt.Println("Mutation: one OTP key consumed")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !matches {
		return fmt.Errorf("unexpected key bundle OTP state: has_expected_otp=%t remaining=%d", hasExpectedOTP, remaining)
	}
	return nil
}

func (s *steps) e2eeKeyBundleShouldOmitOTPAndServerShouldHaveOTPKeysForDevice(expectedCount int, deviceIDText string) error {
	start := time.Now()
	fmt.Println("Given: OTP pool may be empty while identity and SPK still exist")
	fmt.Printf("Input: expected_remaining_count=%d device_id=%s\n", expectedCount, deviceIDText)
	fmt.Println("Action: decode key bundle response and inspect OTP repository")

	bundle, err := decodeKeyBundle(s.ResponseBody)
	if err != nil {
		return err
	}
	deviceID, err := shared.ParseDeviceID(deviceIDText)
	if err != nil {
		return err
	}
	remaining := s.deps.OTPCount(s.accountBDD.LastSessionUserID(), deviceID)
	matches := bundle.IdentityKey != "" && bundle.SignedPreKey != "" && bundle.OTPPreKey == nil && bundle.OTPPreKeyID == nil && remaining == expectedCount

	fmt.Printf("Output: identity_present=%t spk_present=%t otp_present=%t remaining_count=%d match=%t\n",
		bundle.IdentityKey != "", bundle.SignedPreKey != "", bundle.OTPPreKey != nil, remaining, matches)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !matches {
		return fmt.Errorf("expected key bundle without OTP and remaining=%d, got remaining=%d", expectedCount, remaining)
	}
	return nil
}

func (s *steps) e2eeEndpointRequest(endpoint, deviceID string) (method string, path string, payload map[string]any, err error) {
	userID := s.accountBDD.LastSessionUserID()
	switch endpoint {
	case "identity-key":
		return http.MethodPost, "/api/e2ee/identity-key", map[string]any{
			"device_id":       deviceID,
			"public_key":      namedKeyMaterial("alpha", 32),
			"sign_public_key": namedSignKeyMaterial("alpha", 32),
		}, nil
	case "signed-prekey":
		return http.MethodPost, "/api/e2ee/signed-prekey", map[string]any{
			"device_id":  deviceID,
			"key_id":     1,
			"public_key": namedSignedPreKeyMaterial("alpha", 32),
			"signature":  namedSignatureMaterial("alpha", 64),
		}, nil
	case "otp-prekeys":
		return http.MethodPost, "/api/e2ee/otp-prekeys", map[string]any{
			"device_id": deviceID,
			"keys": []map[string]any{{
				"key_id":     1,
				"public_key": otpKeyMaterial(1, 32),
			}},
		}, nil
	case "otp-prekeys-count":
		return http.MethodGet, fmt.Sprintf("/api/e2ee/otp-prekeys/count?device_id=%s", deviceID), nil, nil
	case "key-bundle":
		if userID == 0 {
			return "", "", nil, fmt.Errorf("no logged-in user id available for key-bundle")
		}
		return http.MethodGet, fmt.Sprintf("/api/e2ee/key-bundle/%d?device_id=%s", userID, deviceID), nil, nil
	case "key-status":
		if userID == 0 {
			return "", "", nil, fmt.Errorf("no logged-in user id available for key-status")
		}
		return http.MethodGet, fmt.Sprintf("/api/e2ee/key-status/%d?device_id=%s", userID, deviceID), nil, nil
	case "key-policy":
		return http.MethodGet, "/api/e2ee/key-policy", nil, nil
	case "sender-key":
		return http.MethodPost, "/api/e2ee/sender-key", map[string]any{
			"room_id":              1,
			"sender_member_id":     101,
			"receiver_user_id":     202,
			"sender_key_version":   1,
			"distribution_message": base64.StdEncoding.EncodeToString([]byte("distribution-message")),
		}, nil
	case "sender-keys":
		return http.MethodGet, "/api/e2ee/sender-keys/1", nil, nil
	case "sender-key-distributions":
		return http.MethodGet, "/api/e2ee/sender-key-distributions/1", nil, nil
	case "sender-key-distributions-pending":
		return http.MethodGet, "/api/e2ee/sender-key-distributions/1/pending", nil, nil
	case "sender-key-distributions-consume":
		return http.MethodPost, "/api/e2ee/sender-key-distributions/1/consume", map[string]any{
			"status": "consumed",
		}, nil
	case "sender-key-request":
		return http.MethodPost, "/api/e2ee/sender-key-request", map[string]any{
			"room_id":            1,
			"provider_user_id":   202,
			"provider_device_id": "00000000-0000-0000-0000-000000000202",
			"sender_member_id":   202,
		}, nil
	default:
		return "", "", nil, fmt.Errorf("unsupported E2EE endpoint %q", endpoint)
	}
}

func (s *steps) e2eeOTPReplenishEventShouldBeQueuedForDevice(deviceID string) error {
	start := time.Now()
	fmt.Println("Given: OTP count is below the replenish threshold after key-bundle consumption")
	fmt.Printf("Input: expected_msg_type=%s expected_device_id=%s\n", deliveryqueue.PayloadTypeReplenishOTP, deviceID)
	fmt.Println("Action: inspect fake delivery dispatcher")

	userID, msgType, payload, calls := s.deps.LastDispatch()
	var body struct {
		UserID   int64  `json:"user_id"`
		DeviceID string `json:"device_id"`
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &body); err != nil {
			return err
		}
	}
	expectedUserID := s.accountBDD.LastSessionUserID()
	matches := calls == 1 &&
		userID == expectedUserID &&
		msgType == string(deliveryqueue.PayloadTypeReplenishOTP) &&
		body.UserID == int64(expectedUserID) &&
		body.DeviceID == deviceID

	fmt.Printf("Output: dispatch_calls=%d msg_type=%s user_id=%d device_id=%s match=%t\n",
		calls, msgType, userID, body.DeviceID, matches)
	fmt.Println("Mutation: one replenish delivery event recorded")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !matches {
		return fmt.Errorf("expected one OTP replenish event for user=%d device=%s, got calls=%d msg_type=%s user=%d device=%s",
			expectedUserID, deviceID, calls, msgType, userID, body.DeviceID)
	}
	return nil
}

func (s *steps) noE2EEOTPReplenishEventShouldBeQueued() error {
	start := time.Now()
	fmt.Println("Given: OTP count is not below the replenish threshold after key-bundle consumption")
	fmt.Println("Input: expected_dispatch_calls=0")
	fmt.Println("Action: inspect fake delivery dispatcher")

	_, msgType, _, calls := s.deps.LastDispatch()
	fmt.Printf("Output: dispatch_calls=%d msg_type=%s no_event=%t\n", calls, msgType, calls == 0)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if calls != 0 {
		return fmt.Errorf("expected no OTP replenish dispatch, got calls=%d msg_type=%s", calls, msgType)
	}
	return nil
}

func (s *steps) authorizationHeader() (string, error) {
	if s.Response == nil {
		if s.authHeader != "" {
			return s.authHeader, nil
		}
		return "", fmt.Errorf("no login response available")
	}
	if header := s.Response.Header.Get("Authorization"); header != "" {
		if !strings.HasPrefix(header, "Bearer ") {
			header = "Bearer " + header
		}
		s.authHeader = header
		return header, nil
	}
	if s.authHeader != "" {
		return s.authHeader, nil
	}
	return "", fmt.Errorf("no Authorization header from latest response")
}

func (s *steps) authHeaders(authHeader, deviceID string) map[string]string {
	if deviceID == "" {
		deviceID = s.accountBDD.LastSessionDeviceID().String()
	}
	return map[string]string{
		"Authorization": authHeader,
		"X-Device-ID":   deviceID,
	}
}

func (s *steps) doJSONRequestWithHeaders(method, path string, payload any, headers map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, s.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	s.Response = resp
	s.ResponseBody, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return nil
}

func decodeKeyStatus(body []byte) (keyStatusResponse, error) {
	var status keyStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return status, err
	}
	return status, nil
}

func decodeKeyBundle(body []byte) (keyBundleResponse, error) {
	var bundle keyBundleResponse
	if err := json.Unmarshal(body, &bundle); err != nil {
		return bundle, err
	}
	return bundle, nil
}

func namedKeyMaterial(name string, length int) string {
	return base64.StdEncoding.EncodeToString(repeatedKeyByte(seedForName(name), length))
}

func namedSignKeyMaterial(name string, length int) string {
	return base64.StdEncoding.EncodeToString(repeatedKeyByte(seedForName(name)+1, length))
}

func namedSignedPreKeyMaterial(name string, length int) string {
	return base64.StdEncoding.EncodeToString(repeatedKeyByte(seedForName(name)+2, length))
}

func namedSignatureMaterial(name string, length int) string {
	return base64.StdEncoding.EncodeToString(repeatedKeyByte(seedForName(name)+3, length))
}

func otpKeyMaterial(keyID int, length int) string {
	return base64.StdEncoding.EncodeToString(repeatedKeyByte(byte(100+keyID), length))
}

func shortKeyMaterial(length int) string {
	return base64.StdEncoding.EncodeToString(repeatedKeyByte(9, length))
}

func repeatedKeyByte(value byte, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		out[i] = value
	}
	return out
}

func seedForName(name string) byte {
	switch name {
	case "alpha":
		return 1
	case "beta":
		return 20
	default:
		return 40
	}
}

// ─── Sender key request steps ─────────────────────────────────────────────────

func (s *steps) e2eeKeysAreBootstrappedForTheLoggedInUser() error {
	s.start = time.Now()
	userID := s.accountBDD.LastSessionUserID()
	if userID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	fmt.Println("Given: E2EE keys are bootstrapped for the logged in user")
	fmt.Printf("Input: user_id=%d\n", userID)
	fmt.Println("Action: seed caller participant in SKR deps")
	// The caller participant is seeded lazily by each room member setup step.
	fmt.Printf("Output: caller_user_id=%d\n", userID)
	fmt.Println("Mutation: none yet")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) roomMemberSetupSameRoomNoKey(roomID, callerMemberID, providerMemberID int64) error {
	return s.roomMemberSetup(roomID, callerMemberID, providerMemberID, roomID, false, false, false)
}

func (s *steps) roomMemberSetupSameRoomWithAvailableDistribution(roomID, callerMemberID, providerMemberID int64) error {
	return s.roomMemberSetup(roomID, callerMemberID, providerMemberID, roomID, true, true, false)
}

func (s *steps) roomMemberSetupSameRoomWithAvailableDistributionAndPendingRequest(roomID, callerMemberID, providerMemberID int64) error {
	return s.roomMemberSetup(roomID, callerMemberID, providerMemberID, roomID, true, true, true)
}

func (s *steps) roomMemberSetupDifferentRoom(roomID, callerMemberID, providerMemberID int64) error {
	return s.roomMemberSetup(roomID, callerMemberID, providerMemberID, roomID+100, false, false, false)
}

func (s *steps) roomMemberSetupCallerNotInRoom(roomID, providerMemberID int64) error {
	s.start = time.Now()
	callerUserID := s.accountBDD.LastSessionUserID()
	if callerUserID == 0 {
		return fmt.Errorf("no logged in user available for room member setup")
	}

	callerPID := participant.ID(roomID + 5000)
	providerUID := shared.UserID(providerMemberID + 2000)
	providerPID := participant.ID(providerMemberID + 1000)

	s.deps.SKR.SeedParticipant(callerUserID, callerPID)
	s.deps.SKR.SeedMember(providerUID, providerPID, chatmember.ID(providerMemberID), chatroom.ID(roomID))

	fmt.Println("Given: provider membership exists but caller is not a member of the room")
	fmt.Printf("Input: room_id=%d provider_member=%d caller_user_id=%d\n", roomID, providerMemberID, callerUserID)
	fmt.Println("Action: seed caller participant without room membership and seed provider room membership")
	fmt.Printf("Output: caller_participant_seeded=true provider_member_seeded=true\n")
	fmt.Println("Mutation: caller participant seeded without chat membership")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) roomMemberSetup(
	roomID, callerMemberID, providerMemberID, providerRoomID int64,
	seedProviderKey bool,
	seedAvailableDistribution bool,
	seedPendingRequest bool,
) error {
	s.start = time.Now()
	callerUserID := s.accountBDD.LastSessionUserID()
	if callerUserID == 0 {
		return fmt.Errorf("no logged in user available for room member setup")
	}
	// Use synthetic user IDs derived from member IDs for the provider.
	callerPID := participant.ID(callerMemberID + 1000)
	providerUID := shared.UserID(providerMemberID + 2000)
	providerPID := participant.ID(providerMemberID + 1000)
	providerDeviceID := defaultReadyAccountDevice(shared.AccountID(providerUID), providerUID).DeviceID

	s.deps.SKR.SeedMember(callerUserID, callerPID, chatmember.ID(callerMemberID), chatroom.ID(roomID))
	s.deps.SKR.SeedMember(providerUID, providerPID, chatmember.ID(providerMemberID), chatroom.ID(providerRoomID))
	callerDeviceID := s.accountBDD.LastSessionDeviceID()
	if seedProviderKey {
		s.deps.SKR.SeedProviderKeyVersionForDevice(chatmember.ID(providerMemberID), providerDeviceID, 99)
	}
	if seedAvailableDistribution {
		s.deps.SKR.SeedDistributionForPair(
			roomID,
			chatmember.ID(providerMemberID),
			providerDeviceID,
			chatmember.ID(callerMemberID),
			callerDeviceID,
			99,
			senderkeydistribution.StatusAvailable,
		)
	}
	if seedPendingRequest {
		s.deps.SKR.SeedPendingSKRRequestForPair(chatmember.ID(callerMemberID), callerDeviceID, chatmember.ID(providerMemberID), providerDeviceID)
	}
	fmt.Println("Given: room member setup complete")
	fmt.Printf("Input: room_id=%d caller_member=%d provider_member=%d provider_room=%d with_key=%t with_available_distribution=%t with_pending_request=%t\n",
		roomID, callerMemberID, providerMemberID, providerRoomID, seedProviderKey, seedAvailableDistribution, seedPendingRequest)
	fmt.Println("Mutation: SKR participants and members seeded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) callerAndProviderAreBlocked(callerMemberID, providerMemberID int64) error {
	s.start = time.Now()
	callerMember, err := s.deps.SKR.chatMemberRepo.FindByID(nil, chatmember.ID(callerMemberID))
	if err != nil {
		return err
	}
	providerMember, err := s.deps.SKR.chatMemberRepo.FindByID(nil, chatmember.ID(providerMemberID))
	if err != nil {
		return err
	}
	callerParticipant, err := s.deps.SKR.participantRepo.FindByID(nil, callerMember.ParticipantID)
	if err != nil || callerParticipant.UserID == nil {
		return fmt.Errorf("caller participant not found for member %d", callerMemberID)
	}
	providerParticipant, err := s.deps.SKR.participantRepo.FindByID(nil, providerMember.ParticipantID)
	if err != nil || providerParticipant.UserID == nil {
		return fmt.Errorf("provider participant not found for member %d", providerMemberID)
	}
	s.deps.SKR.SeedBlockedFriendship(*callerParticipant.UserID, *providerParticipant.UserID)
	fmt.Println("Given: caller and provider have a blocked relationship")
	fmt.Printf("Input: caller_member_id=%d provider_member_id=%d\n", callerMemberID, providerMemberID)
	fmt.Println("Mutation: friendship status seeded as blocked")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) senderKeyProviderSetup(roomID, providerMemberID, receiverMemberID int64) error {
	s.start = time.Now()
	providerUserID := s.accountBDD.LastSessionUserID()
	if providerUserID == 0 {
		return fmt.Errorf("no logged in user available for sender key provider setup")
	}

	providerPID := participant.ID(providerMemberID + 1000)
	receiverUID := shared.UserID(receiverMemberID + 2000)
	receiverPID := participant.ID(receiverMemberID + 1000)

	s.deps.SKR.SeedMember(providerUserID, providerPID, chatmember.ID(providerMemberID), chatroom.ID(roomID))
	s.deps.SKR.SeedMember(receiverUID, receiverPID, chatmember.ID(receiverMemberID), chatroom.ID(roomID))

	fmt.Println("Given: authenticated user is the sender key provider in the room")
	fmt.Printf("Input: room_id=%d provider_member_id=%d receiver_member_id=%d\n", roomID, providerMemberID, receiverMemberID)
	fmt.Println("Mutation: provider and receiver memberships seeded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) senderKeyReceiverSetupWithAvailableDistribution(roomID, senderMemberID, receiverMemberID, version int64) error {
	s.start = time.Now()
	receiverUserID := s.accountBDD.LastSessionUserID()
	if receiverUserID == 0 {
		return fmt.Errorf("no logged in user available for sender key receiver setup")
	}

	senderUID := shared.UserID(senderMemberID + 2000)
	senderPID := participant.ID(senderMemberID + 1000)
	receiverPID := participant.ID(receiverMemberID + 1000)
	senderDeviceID := defaultReadyAccountDevice(shared.AccountID(senderUID), senderUID).DeviceID

	s.deps.SKR.SeedMember(senderUID, senderPID, chatmember.ID(senderMemberID), chatroom.ID(roomID))
	s.deps.SKR.SeedMember(receiverUserID, receiverPID, chatmember.ID(receiverMemberID), chatroom.ID(roomID))
	s.deps.SKR.SeedProviderKeyVersionForDevice(chatmember.ID(senderMemberID), senderDeviceID, version)
	receiverDeviceID := s.accountBDD.LastSessionDeviceID()
	s.lastDistributionID = int64(s.deps.SKR.SeedDistributionForPair(
		roomID,
		chatmember.ID(senderMemberID),
		senderDeviceID,
		chatmember.ID(receiverMemberID),
		receiverDeviceID,
		version,
		senderkeydistribution.StatusAvailable,
	))

	fmt.Println("Given: authenticated user is the pending distribution receiver in the room")
	fmt.Printf("Input: room_id=%d sender_member_id=%d receiver_member_id=%d version=%d distribution_id=%d\n", roomID, senderMemberID, receiverMemberID, version, s.lastDistributionID)
	fmt.Println("Mutation: sender key metadata and available distribution seeded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iProvideASenderKeyDistribution(roomID, receiverMemberID, version int64) error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}
	senderMemberID, err := s.currentLoggedInMemberIDInRoom(chatroom.ID(roomID))
	if err != nil {
		return err
	}
	receiverUserID, err := s.userIDForMember(chatmember.ID(receiverMemberID))
	if err != nil {
		return err
	}

	fmt.Println("Given: authenticated provider uploads a sealed sender key distribution")
	fmt.Printf("Input: room_id=%d sender_member_id=%d receiver_user_id=%d sender_key_version=%d\n", roomID, senderMemberID, receiverUserID, version)
	fmt.Println("Action: POST /api/e2ee/sender-key")

	payload := map[string]any{
		"room_id":              roomID,
		"sender_member_id":     int64(senderMemberID),
		"receiver_user_id":     int64(receiverUserID),
		"sender_key_version":   version,
		"distribution_message": base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("dist-%d", version))),
	}
	if err := s.doE2EEJSONRequest(http.MethodPost, "/api/e2ee/sender-key", payload, authHeader); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d\n", s.Response.StatusCode)
	fmt.Println("Mutation: sender_key_distributions may be updated")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iRequestSenderKeyDistributionStatus(roomID int64) error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}

	fmt.Println("Given: authenticated room member wants sender key room summary")
	fmt.Printf("Input: room_id=%d\n", roomID)
	fmt.Println("Action: GET /api/e2ee/sender-key-distributions/:room_id")

	if err := s.DoRequestWithHeaders(http.MethodGet, fmt.Sprintf("/api/e2ee/sender-key-distributions/%d", roomID), s.authHeaders(authHeader, "")); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d\n", s.Response.StatusCode)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) senderKeyDistributionStatusShouldShowOwnKeyExists(expected string) error {
	start := time.Now()
	fmt.Println("Given: sender key distribution room summary response is available")
	fmt.Printf("Input: expected_own_sender_key_exists=%s\n", expected)

	var body senderKeyDistributionStatusResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	expectedValue := expected == "true"
	if body.OwnMemberSenderKeyExists != expectedValue {
		return fmt.Errorf("expected own_member_sender_key_exists=%t, got %t", expectedValue, body.OwnMemberSenderKeyExists)
	}
	fmt.Printf("Output: own_member_sender_key_exists=%t\n", body.OwnMemberSenderKeyExists)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func containsMemberRef(refs []senderKeyDeviceRefStep, memberID int64) bool {
	for _, ref := range refs {
		if ref.MemberID == memberID {
			return true
		}
	}
	return false
}

func containsMemberDeviceRef(refs []senderKeyDeviceRefStep, memberID int64, deviceID string) bool {
	for _, ref := range refs {
		if ref.MemberID == memberID && ref.DeviceID == deviceID {
			return true
		}
	}
	return false
}

func (s *steps) senderKeyDistributionStatusShouldListAvailableSenderMember(memberID int64) error {
	start := time.Now()
	var body senderKeyDistributionStatusResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	if containsMemberRef(body.AvailableFromSources, memberID) {
		fmt.Printf("Output: available_from_sources=%v\n", body.AvailableFromSources)
		fmt.Printf("Duration: %s\n", time.Since(start))
		return nil
	}
	return fmt.Errorf("expected available_from_sources to include member %d, got %v", memberID, body.AvailableFromSources)
}

func (s *steps) senderKeyDistributionStatusShouldListAvailableSenderMemberDevice(memberID int64, deviceID string) error {
	start := time.Now()
	var body senderKeyDistributionStatusResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	if containsMemberDeviceRef(body.AvailableFromSources, memberID, deviceID) {
		fmt.Printf("Output: available_from_sources=%v\n", body.AvailableFromSources)
		fmt.Printf("Duration: %s\n", time.Since(start))
		return nil
	}
	return fmt.Errorf("expected available_from_sources to include member %d device %s, got %v", memberID, deviceID, body.AvailableFromSources)
}

func (s *steps) senderKeyDistributionStatusShouldListAvailableReceiverMember(memberID int64) error {
	start := time.Now()
	var body senderKeyDistributionStatusResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	if containsMemberRef(body.AvailableToTargets, memberID) {
		fmt.Printf("Output: available_to_targets=%v\n", body.AvailableToTargets)
		fmt.Printf("Duration: %s\n", time.Since(start))
		return nil
	}
	return fmt.Errorf("expected available_to_targets to include member %d, got %v", memberID, body.AvailableToTargets)
}

func (s *steps) senderKeyDistributionStatusShouldListAvailableReceiverMemberDevice(memberID int64, deviceID string) error {
	start := time.Now()
	var body senderKeyDistributionStatusResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	if containsMemberDeviceRef(body.AvailableToTargets, memberID, deviceID) {
		fmt.Printf("Output: available_to_targets=%v\n", body.AvailableToTargets)
		fmt.Printf("Duration: %s\n", time.Since(start))
		return nil
	}
	return fmt.Errorf("expected available_to_targets to include member %d device %s, got %v", memberID, deviceID, body.AvailableToTargets)
}

func (s *steps) senderKeyDistributionStatusShouldListPendingReceiverMember(memberID int64) error {
	start := time.Now()
	var body senderKeyDistributionStatusResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	if containsMemberRef(body.PendingReceivers, memberID) {
		fmt.Printf("Output: pending_receivers=%v\n", body.PendingReceivers)
		fmt.Printf("Duration: %s\n", time.Since(start))
		return nil
	}
	return fmt.Errorf("expected pending_receivers to include %d, got %v", memberID, body.PendingReceivers)
}

func (s *steps) senderKeyDistributionStatusShouldListPendingReceiverMemberDevice(memberID int64, deviceID string) error {
	start := time.Now()
	var body senderKeyDistributionStatusResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	if containsMemberDeviceRef(body.PendingReceivers, memberID, deviceID) {
		fmt.Printf("Output: pending_receivers=%v\n", body.PendingReceivers)
		fmt.Printf("Duration: %s\n", time.Since(start))
		return nil
	}
	return fmt.Errorf("expected pending_receivers to include member %d device %s, got %v", memberID, deviceID, body.PendingReceivers)
}

func (s *steps) iListPendingSenderKeyDistributions(roomID int64) error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}

	fmt.Println("Given: authenticated receiver wants pending sender key distributions")
	fmt.Printf("Input: room_id=%d\n", roomID)
	fmt.Println("Action: GET /api/e2ee/sender-key-distributions/:room_id/pending")

	if err := s.DoRequestWithHeaders(http.MethodGet, fmt.Sprintf("/api/e2ee/sender-key-distributions/%d/pending", roomID), s.authHeaders(authHeader, "")); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d\n", s.Response.StatusCode)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) pendingSenderKeyDistributionsShouldInclude(senderMemberID, receiverMemberID, version int64) error {
	start := time.Now()
	var body pendingSenderKeyDistributionsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, dist := range body.Distributions {
		if dist.SenderMemberID == senderMemberID && dist.ReceiverMemberID == receiverMemberID && dist.SenderKeyVersion == version {
			s.lastDistributionID = dist.DistributionID
			fmt.Printf("Output: found_distribution_id=%d distributions=%d\n", dist.DistributionID, len(body.Distributions))
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf("expected pending distribution sender=%d receiver=%d version=%d, got %+v", senderMemberID, receiverMemberID, version, body.Distributions)
}

func (s *steps) pendingSenderKeyDistributionsShouldIncludeDevices(senderMemberID int64, senderDeviceID string, receiverMemberID int64, receiverDeviceID string, version int64) error {
	start := time.Now()
	var body pendingSenderKeyDistributionsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, dist := range body.Distributions {
		if dist.SenderMemberID == senderMemberID &&
			dist.SenderDeviceID == senderDeviceID &&
			dist.ReceiverMemberID == receiverMemberID &&
			dist.ReceiverDeviceID == receiverDeviceID &&
			dist.SenderKeyVersion == version {
			s.lastDistributionID = dist.DistributionID
			fmt.Printf("Output: found_distribution_id=%d distributions=%d\n", dist.DistributionID, len(body.Distributions))
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf(
		"expected pending distribution sender=%d sender_device=%s receiver=%d receiver_device=%s version=%d, got %+v",
		senderMemberID,
		senderDeviceID,
		receiverMemberID,
		receiverDeviceID,
		version,
		body.Distributions,
	)
}

func (s *steps) iMarkTheFirstPendingSenderKeyDistributionAs(status string) error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}
	if s.lastDistributionID == 0 {
		return fmt.Errorf("no pending distribution id recorded")
	}

	fmt.Println("Given: authenticated receiver processes the pending distribution")
	fmt.Printf("Input: distribution_id=%d status=%s\n", s.lastDistributionID, status)
	fmt.Println("Action: POST /api/e2ee/sender-key-distributions/:distribution_id/consume")

	payload := map[string]any{"status": status}
	if err := s.doE2EEJSONRequest(http.MethodPost, fmt.Sprintf("/api/e2ee/sender-key-distributions/%d/consume", s.lastDistributionID), payload, authHeader); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d\n", s.Response.StatusCode)
	fmt.Println("Mutation: distribution status may change")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theFirstPendingSenderKeyDistributionShouldNowBe(status string) error {
	start := time.Now()
	dist, ok := s.deps.SKR.FindDistributionByID(senderkeydistribution.ID(s.lastDistributionID))
	if !ok {
		return fmt.Errorf("expected distribution id %d to exist", s.lastDistributionID)
	}
	if string(dist.Status) != status {
		return fmt.Errorf("expected distribution id %d to be %s, got %s", s.lastDistributionID, status, dist.Status)
	}
	fmt.Printf("Output: distribution_id=%d status=%s\n", s.lastDistributionID, dist.Status)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) senderKeyNeededEventShouldHaveBeenBroadcast(providerMemberID int64) error {
	start := time.Now()
	fmt.Println("Given: consume failed and a sender key request was requeued")
	fmt.Printf("Input: expected_provider_member_id=%d\n", providerMemberID)
	fmt.Println("Action: inspect recording broadcaster for e2ee.sender_key_needed events")

	// The goroutine in the consume use case fires asynchronously; give it a moment.
	time.Sleep(50 * time.Millisecond)

	messages := s.deps.SKR.BroadcastMessages()
	for _, msg := range messages {
		var envelope struct {
			Type    string `json:"type"`
			Payload struct {
				SenderMemberID int64 `json:"sender_member_id"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
			continue
		}
		if envelope.Type == "e2ee.sender_key_needed" && envelope.Payload.SenderMemberID == providerMemberID {
			fmt.Printf("Output: found e2ee.sender_key_needed for provider_member_id=%d\n", providerMemberID)
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}

	return fmt.Errorf("expected e2ee.sender_key_needed event for provider member %d, got %d broadcast messages", providerMemberID, len(messages))
}

func (s *steps) defaultReadyDeviceIDForMember(memberID chatmember.ID) (shared.DeviceID, error) {
	member, err := s.deps.SKR.chatMemberRepo.FindByID(nil, memberID)
	if err != nil {
		return shared.DeviceID{}, err
	}
	participantData, err := s.deps.SKR.participantRepo.FindByID(nil, member.ParticipantID)
	if err != nil || participantData.UserID == nil {
		return shared.DeviceID{}, fmt.Errorf("participant user not found for member %d", memberID)
	}
	return defaultReadyAccountDevice(shared.AccountID(*participantData.UserID), *participantData.UserID).DeviceID, nil
}

func (s *steps) userIDForMember(memberID chatmember.ID) (shared.UserID, error) {
	member, err := s.deps.SKR.chatMemberRepo.FindByID(nil, memberID)
	if err != nil {
		return 0, err
	}
	participantData, err := s.deps.SKR.participantRepo.FindByID(nil, member.ParticipantID)
	if err != nil || participantData.UserID == nil {
		return 0, fmt.Errorf("participant user not found for member %d", memberID)
	}
	return *participantData.UserID, nil
}

func (s *steps) currentLoggedInMemberIDInRoom(roomID chatroom.ID) (chatmember.ID, error) {
	userID := s.accountBDD.LastSessionUserID()
	if userID == 0 {
		return 0, fmt.Errorf("no logged in user available for sender member lookup")
	}
	participantData, err := s.deps.SKR.participantRepo.FindByUserID(context.Background(), userID)
	if err != nil {
		return 0, err
	}
	member, err := s.deps.SKR.chatMemberRepo.FindByRoomAndParticipant(context.Background(), roomID, participantData.ID)
	if err != nil {
		return 0, err
	}
	return member.ID, nil
}

func (s *steps) iCreateASenderKeyRequestForRoomAndProviderMember(roomID, providerMemberID int64) error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}
	providerUserID, err := s.userIDForMember(chatmember.ID(providerMemberID))
	if err != nil {
		return err
	}
	providerDeviceID, err := s.defaultReadyDeviceIDForMember(chatmember.ID(providerMemberID))
	if err != nil {
		return err
	}
	fmt.Println("Given: authenticated user creates a sender key request")
	fmt.Printf("Input: room_id=%d provider_user_id=%d sender_member_id=%d provider_device_id=%s\n", roomID, providerUserID, providerMemberID, providerDeviceID.String())
	fmt.Println("Action: POST /api/e2ee/sender-key-request")

	payload := map[string]any{
		"room_id":            roomID,
		"provider_user_id":   int64(providerUserID),
		"provider_device_id": providerDeviceID.String(),
		"sender_member_id":   providerMemberID,
	}
	if err := s.doE2EEJSONRequest(http.MethodPost, "/api/e2ee/sender-key-request", payload, authHeader); err != nil {
		return err
	}
	fmt.Printf("Output: status=%d\n", s.Response.StatusCode)
	fmt.Println("Mutation: sender_key_requests may be updated")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) aSenderKeyRequestRowShouldExist(requesterMemberID, providerMemberID int64) error {
	start := time.Now()
	fmt.Println("Given: a sender key request row should have been persisted")
	fmt.Printf("Input: requester_member_id=%d provider_member_id=%d\n", requesterMemberID, providerMemberID)

	_, ok := s.deps.SKR.FindSKRRequest(chatmember.ID(requesterMemberID), chatmember.ID(providerMemberID))
	if !ok {
		return fmt.Errorf("expected sender key request from member %d to provider %d, but none was found", requesterMemberID, providerMemberID)
	}
	fmt.Println("Output: sender_key_request_found=true")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) noSenderKeyRequestRowShouldExist(requesterMemberID, providerMemberID int64) error {
	start := time.Now()
	fmt.Println("Given: no sender key request row should have been persisted")
	fmt.Printf("Input: requester_member_id=%d provider_member_id=%d\n", requesterMemberID, providerMemberID)

	_, ok := s.deps.SKR.FindSKRRequest(chatmember.ID(requesterMemberID), chatmember.ID(providerMemberID))
	if ok {
		return fmt.Errorf("expected no sender key request from member %d to provider %d, but one was found", requesterMemberID, providerMemberID)
	}
	fmt.Println("Output: sender_key_request_found=false")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) pendingSenderKeyRequestCountFromMemberToProviderShouldBe(requesterMemberID, providerMemberID int64, expected int) error {
	start := time.Now()
	fmt.Println("Given: sender key requests should remain idempotent per requester/provider pair")
	fmt.Printf("Input: requester_member_id=%d provider_member_id=%d expected_pending=%d\n", requesterMemberID, providerMemberID, expected)
	fmt.Println("Action: inspect in-memory sender_key_requests state")

	actual := s.deps.SKR.PendingSKRCount(chatmember.ID(requesterMemberID), chatmember.ID(providerMemberID))
	fmt.Printf("Output: actual_pending=%d\n", actual)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if actual != int(expected) {
		return fmt.Errorf("expected %d pending sender key requests from member %d to provider %d, got %d", expected, requesterMemberID, providerMemberID, actual)
	}
	return nil
}

func (s *steps) e2eeAuthorizationHeader() (string, error) {
	return s.authorizationHeader()
}

func (s *steps) aSelfSenderKeySyncExistsForTheLoggedInUserWithoutProvider(participantID int64, requesterDeviceIDText, status string) error {
	return s.seedSelfSenderKeySync(participantID, requesterDeviceIDText, "", status)
}

func (s *steps) aSelfSenderKeySyncExistsForTheLoggedInUserWithProvider(participantID int64, requesterDeviceIDText, providerDeviceIDText, status string) error {
	return s.seedSelfSenderKeySync(participantID, requesterDeviceIDText, providerDeviceIDText, status)
}

func (s *steps) seedSelfSenderKeySync(participantID int64, requesterDeviceIDText, providerDeviceIDText, status string) error {
	s.start = time.Now()
	userID := s.accountBDD.LastSessionUserID()
	if userID == 0 {
		return fmt.Errorf("no logged in user available for self sender key sync setup")
	}

	requesterDeviceID, err := shared.ParseDeviceID(requesterDeviceIDText)
	if err != nil {
		return err
	}

	var providerDeviceID *shared.DeviceID
	if providerDeviceIDText != "" {
		parsed, parseErr := shared.ParseDeviceID(providerDeviceIDText)
		if parseErr != nil {
			return parseErr
		}
		providerDeviceID = &parsed
	}

	s.deps.SKR.SeedParticipant(userID, participant.ID(participantID))
	s.deps.SKR.SeedSelfSenderKeySync(&selfsenderkeysync.SelfSenderKeySync{
		ParticipantID:     participant.ID(participantID),
		RequesterDeviceID: requesterDeviceID,
		ProviderDeviceID:  providerDeviceID,
		Status:            selfsenderkeysync.Status(status),
		RequestedAt:       time.Now().Add(-time.Minute),
		UpdatedAt:         time.Now().Add(-time.Minute),
	})

	if providerDeviceID != nil {
		fmt.Println("Given: a self sender key sync exists with both requester and provider devices")
		fmt.Printf("Input: participant_id=%d requester_device_id=%s provider_device_id=%s status=%s\n", participantID, requesterDeviceIDText, providerDeviceIDText, status)
	} else {
		fmt.Println("Given: a self sender key sync exists with only a requester device")
		fmt.Printf("Input: participant_id=%d requester_device_id=%s status=%s\n", participantID, requesterDeviceIDText, status)
	}
	fmt.Println("Action: seed participant_self_sender_key_syncs fake state")
	fmt.Printf("Output: self_sync_seeded=true provider_present=%t\n", providerDeviceID != nil)
	fmt.Println("Mutation: self sender key sync fake repository expanded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theSelfSenderKeySyncRequesterDeviceIsBoundAs(deviceIDText, statusText string) error {
	return s.seedSelfSenderKeySyncAccountBinding(deviceIDText, statusText, true)
}

func (s *steps) theSelfSenderKeySyncProviderDeviceIsBoundAs(deviceIDText, statusText string) error {
	return s.seedSelfSenderKeySyncAccountBinding(deviceIDText, statusText, false)
}

func (s *steps) seedSelfSenderKeySyncAccountBinding(deviceIDText, statusText string, requester bool) error {
	s.start = time.Now()
	accountID := s.accountBDD.LastSessionAccountID()
	if accountID == 0 {
		return fmt.Errorf("no logged in user available for self sender key sync binding setup")
	}
	deviceID, err := shared.ParseDeviceID(deviceIDText)
	if err != nil {
		return err
	}
	status := domainaccount.DeviceStatus(statusText)
	switch status {
	case domainaccount.DeviceStatusPendingVerification, domainaccount.DeviceStatusPendingSync, domainaccount.DeviceStatusSyncing, domainaccount.DeviceStatusReady:
	default:
		return fmt.Errorf("unknown device status %q", statusText)
	}

	roleLabel := "provider"
	if requester {
		roleLabel = "requester"
	}
	fmt.Printf("Given: the self sender key sync %s device is bound to the account\n", roleLabel)
	fmt.Printf("Input: account_id=%d device_id=%s status=%s\n", accountID, deviceIDText, statusText)
	fmt.Println("Action: seed account and device repositories for self sync snapshots")
	s.deps.SKR.SeedAccountBinding(accountID, deviceID, fmt.Sprintf("%s-device", roleLabel), domaindevice.MacOS, status)
	fmt.Printf("Output: device_bound=true binding_status=%s\n", status)
	fmt.Println("Mutation: accounts_devices fake state expanded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theCurrentLoggedInUserIsAvailableToSenderKeyUseCases() error {
	s.start = time.Now()
	userID := s.accountBDD.LastSessionUserID()
	if userID == 0 {
		return fmt.Errorf("no logged in user available for sender key participant seeding")
	}
	accountID := s.accountBDD.LastSessionAccountID()
	if accountID == 0 {
		return fmt.Errorf("no logged in account available for sender key participant seeding")
	}
	participantID := s.accountBDD.LastSessionParticipantID()
	if participantID == 0 {
		return fmt.Errorf("no logged in participant id available for sender key participant seeding")
	}
	accountData := s.accountBDD.LastSessionAccountSnapshot()
	if accountData == nil {
		return fmt.Errorf("no logged in account snapshot available for sender key participant seeding")
	}

	fmt.Println("Given: the logged in user should be available to sender key use cases")
	fmt.Printf("Input: account_id=%d user_id=%d participant_id=%d device_bindings=%d\n", accountID, userID, participantID, len(accountData.Devices))
	fmt.Println("Action: seed the sender key participant and account repositories from the authenticated login state")
	s.deps.SKR.SeedParticipantForAccount(userID, accountID, participantID)
	for index, binding := range accountData.Devices {
		s.deps.SKR.SeedAccountBinding(accountID, binding.DeviceID, fmt.Sprintf("login-device-%d", index+1), domaindevice.MacOS, binding.Status)
	}
	registeredDevice := s.accountBDD.LastRegisteredAccountDevice()
	if registeredDevice != nil {
		s.deps.SKR.SeedAccountBinding(accountID, registeredDevice.DeviceID, "login-registered-device", domaindevice.MacOS, registeredDevice.Status)
	}
	fmt.Printf("Output: sender_key_participant_seeded=%t account_bindings_seeded=%d registered_device_seeded=%t\n", true, len(accountData.Devices), registeredDevice != nil)
	fmt.Println("Mutation: sender key participant and account repositories expanded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) aSelfSenderKeySyncDistributionExistsForTheLoggedInUser(
	participantID int64,
	roomID int64,
	senderMemberID int64,
	requesterDeviceIDText, providerDeviceIDText string,
	senderKeyVersion int64,
) error {
	return s.seedSelfSenderKeySyncDistribution(
		participantID,
		roomID,
		senderMemberID,
		requesterDeviceIDText,
		providerDeviceIDText,
		senderKeyVersion,
	)
}

func (s *steps) aSelfSenderKeySyncDistributionExistsForTheCurrentLoggedInParticipant(
	roomID int64,
	senderMemberID int64,
	requesterDeviceIDText, providerDeviceIDText string,
	senderKeyVersion int64,
) error {
	userID := s.accountBDD.LastSessionUserID()
	if userID == 0 {
		return fmt.Errorf("no logged in user available for current participant self sender key sync distribution setup")
	}
	currentParticipant, err := s.deps.SKR.participantRepo.FindByUserID(context.Background(), userID)
	if err != nil {
		return err
	}
	return s.seedSelfSenderKeySyncDistribution(
		int64(currentParticipant.ID),
		roomID,
		senderMemberID,
		requesterDeviceIDText,
		providerDeviceIDText,
		senderKeyVersion,
	)
}

func (s *steps) seedSelfSenderKeySyncDistribution(
	participantID int64,
	roomID int64,
	senderMemberID int64,
	requesterDeviceIDText, providerDeviceIDText string,
	senderKeyVersion int64,
) error {
	s.start = time.Now()
	requesterDeviceID, err := shared.ParseDeviceID(requesterDeviceIDText)
	if err != nil {
		return err
	}
	providerDeviceID, err := shared.ParseDeviceID(providerDeviceIDText)
	if err != nil {
		return err
	}

	syncState, err := s.deps.SKR.selfSyncRepo.FindByParticipantID(context.Background(), participant.ID(participantID))
	if err != nil {
		return err
	}
	s.deps.SKR.SeedRoomMember(participant.ID(participantID), chatmember.ID(senderMemberID), chatroom.ID(roomID))
	rowsKey := s.deps.SKR.selfSyncCopyRepo.key(syncState.ID, participant.ID(participantID), requesterDeviceID)
	rows := make([]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, 0)
	for _, row := range s.deps.SKR.selfSyncCopyRepo.rows[rowsKey] {
		rows = append(rows, s.deps.SKR.selfSyncCopyRepo.clone(row))
	}
	rows = append(rows, &selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution{
		SenderMemberID:      chatmember.ID(senderMemberID),
		SenderDeviceID:      providerDeviceID,
		SenderKeyVersion:    senderKeyVersion,
		DistributionMessage: []byte(fmt.Sprintf("self-sync-%d", senderKeyVersion)),
		Status:              selfsenderkeysyncdistribution.StatusAvailable,
		CreatedAt:           time.Now().Add(-time.Minute),
	})
	err = s.deps.SKR.selfSyncCopyRepo.ReplaceForSync(
		context.Background(),
		syncState.ID,
		participant.ID(participantID),
		requesterDeviceID,
		providerDeviceID,
		rows,
	)
	if err != nil {
		return err
	}

	fmt.Println("Given: a pending self sender key sync distribution already exists for the requester")
	fmt.Printf("Input: participant_id=%d room_id=%d sender_member_id=%d requester_device_id=%s provider_device_id=%s version=%d\n",
		participantID, roomID, senderMemberID, requesterDeviceIDText, providerDeviceIDText, senderKeyVersion)
	fmt.Println("Action: seed room membership and self sync copy rows")
	fmt.Println("Output: self_sync_distribution_seeded=true")
	fmt.Println("Mutation: self_sender_key_sync_distributions fake state expanded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) selfSenderKeySyncSnapshotLookupsWillFailAfterMutation() error {
	s.start = time.Now()
	fmt.Println("Given: authoritative self sync snapshot lookups will fail after the mutation commits")
	fmt.Println("Input: account_lookup_failure=true")
	fmt.Println("Action: configure SKR fake account repo to fail FindByID")
	s.deps.SKR.SetSelfSyncSnapshotLookupFailure(true)
	fmt.Println("Output: self_sync_snapshot_lookup_failure=true")
	fmt.Println("Mutation: fake account repository now forces fallback snapshots")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iListPendingSelfSenderKeySyncDistributions() error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: the requester device asks for pending self sync distributions")
	fmt.Println("Action: GET /api/e2ee/self-sender-key-sync/distributions/pending")
	if err := s.DoRequestWithHeaders(http.MethodGet, "/api/e2ee/self-sender-key-sync/distributions/pending", map[string]string{
		"Authorization": authHeader,
		"X-Device-ID":   s.accountBDD.LastSessionDeviceID().String(),
	}); err != nil {
		return err
	}
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) pendingSelfSenderKeySyncDistributionsShouldInclude(senderMemberID int64, senderDeviceID string, version int64) error {
	start := time.Now()
	fmt.Println("Given: a pending self sender key sync distributions response is available")
	fmt.Printf("Input: sender_member_id=%d sender_device_id=%s version=%d\n", senderMemberID, senderDeviceID, version)
	fmt.Println("Action: decode pending self sync distributions response")

	var body pendingSelfSenderKeySyncDistributionsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, dist := range body.Distributions {
		if dist.SenderMemberID == senderMemberID && dist.SenderDeviceID == senderDeviceID && dist.SenderKeyVersion == version {
			s.lastSelfSyncDistributionID = dist.DistributionID
			fmt.Printf("Output: found_distribution_id=%d distributions=%d\n", dist.DistributionID, len(body.Distributions))
			fmt.Println("Mutation: remembered_self_sync_distribution_id=true")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf("expected pending self sync distribution sender=%d sender_device=%s version=%d, got %+v", senderMemberID, senderDeviceID, version, body.Distributions)
}

func (s *steps) iMarkTheFirstPendingSelfSenderKeySyncDistributionAs(status string) error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}
	if s.lastSelfSyncDistributionID == 0 {
		return fmt.Errorf("no self sync distribution id recorded")
	}
	fmt.Println("Given: the requester device processes the first pending self sync distribution")
	fmt.Printf("Input: distribution_id=%d status=%s\n", s.lastSelfSyncDistributionID, status)
	fmt.Println("Action: POST /api/e2ee/self-sender-key-sync/distributions/:distribution_id/consume")
	if err := s.doE2EEJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/api/e2ee/self-sender-key-sync/distributions/%d/consume", s.lastSelfSyncDistributionID),
		map[string]any{"status": status},
		authHeader,
	); err != nil {
		return err
	}
	fmt.Printf("Output: status=%d\n", s.Response.StatusCode)
	fmt.Println("Mutation: self sync distribution status may change")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theFirstPendingSelfSenderKeySyncDistributionShouldNowBe(status string) error {
	start := time.Now()
	if s.lastSelfSyncDistributionID == 0 {
		return fmt.Errorf("no self sync distribution id recorded")
	}

	fmt.Println("Given: the last self sync distribution id should now reflect the consume result")
	fmt.Printf("Input: distribution_id=%d expected_status=%s\n", s.lastSelfSyncDistributionID, status)
	fmt.Println("Action: inspect the self sync distribution fake repository")

	expectedStatus := selfsenderkeysyncdistribution.Status(status)
	for _, rows := range s.deps.SKR.selfSyncCopyRepo.rows {
		if dist, ok := rows[selfsenderkeysyncdistribution.ID(s.lastSelfSyncDistributionID)]; ok {
			fmt.Printf("Output: distribution_id=%d actual_status=%s\n", s.lastSelfSyncDistributionID, dist.Status)
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			if dist.Status != expectedStatus {
				return fmt.Errorf("expected self sync distribution id %d to be %s, got %s", s.lastSelfSyncDistributionID, status, dist.Status)
			}
			return nil
		}
	}
	return fmt.Errorf("expected self sync distribution id %d to exist", s.lastSelfSyncDistributionID)
}

func (s *steps) aSelfSenderKeyReceiptShouldExistFromSenderMemberDeviceToRequesterDeviceWithVersionAndSource(
	senderMemberID int64,
	senderDeviceIDText, requesterDeviceIDText string,
	version int64,
	source string,
) error {
	start := time.Now()
	senderDeviceID, err := shared.ParseDeviceID(senderDeviceIDText)
	if err != nil {
		return err
	}
	requesterDeviceID, err := shared.ParseDeviceID(requesterDeviceIDText)
	if err != nil {
		return err
	}
	expectedSource := senderkeyreceipt.Source(source)
	switch expectedSource {
	case senderkeyreceipt.SourceDistribution, senderkeyreceipt.SourceSelfSync, senderkeyreceipt.SourceSenderKeys:
	default:
		return fmt.Errorf("unknown sender key receipt source %q", source)
	}

	fmt.Println("Given: the requester should have recorded a self sync sender key receipt")
	fmt.Printf("Input: sender_member_id=%d sender_device_id=%s requester_device_id=%s version=%d source=%s\n",
		senderMemberID, senderDeviceIDText, requesterDeviceIDText, version, source)
	fmt.Println("Action: inspect the sender_key_receipts fake repository")

	for _, receipt := range s.deps.SKR.receiptRepo.records {
		if receipt.SenderMemberID == chatmember.ID(senderMemberID) &&
			receipt.SenderDeviceID == senderDeviceID &&
			receipt.ReceiverDeviceID == requesterDeviceID &&
			receipt.SenderKeyVersion == version &&
			receipt.Source == expectedSource {
			fmt.Printf("Output: receiver_member_id=%d source=%s\n", receipt.ReceiverMemberID, receipt.Source)
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf(
		"expected self sync receipt sender=%d sender_device=%s requester_device=%s version=%d source=%s",
		senderMemberID,
		senderDeviceIDText,
		requesterDeviceIDText,
		version,
		source,
	)
}

func (s *steps) iAcceptTheSelfSenderKeySync() error {
	return s.postSelfSenderKeySyncMutation("/api/e2ee/self-sender-key-sync/accept", "the current device tries to become the self sync provider", "self sender key sync may advance to syncing", "")
}

func (s *steps) iAcceptTheSelfSenderKeySyncAsDevice(deviceID string) error {
	return s.postSelfSenderKeySyncMutation("/api/e2ee/self-sender-key-sync/accept", "the selected device tries to become the self sync provider", "self sender key sync may advance to syncing", deviceID)
}

func (s *steps) iMarkTheSelfSenderKeySyncUploaded() error {
	return s.postSelfSenderKeySyncMutation("/api/e2ee/self-sender-key-sync/uploaded", "the provider device finished uploading self sync copies", "self sender key sync may advance to uploaded", "")
}

func (s *steps) iMarkTheSelfSenderKeySyncUploadedAsDevice(deviceID string) error {
	return s.postSelfSenderKeySyncMutation("/api/e2ee/self-sender-key-sync/uploaded", "the selected provider device finished uploading self sync copies", "self sender key sync may advance to uploaded", deviceID)
}

func (s *steps) postSelfSenderKeySyncMutation(path, givenCopy, mutationCopy, deviceID string) error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}
	headers := s.authHeaders(authHeader, deviceID)
	fmt.Printf("Given: %s\n", givenCopy)
	fmt.Printf("Input: device_id=%s\n", headers["X-Device-ID"])
	fmt.Printf("Action: POST %s\n", path)
	if err := s.doJSONRequestWithHeaders(http.MethodPost, path, map[string]any{}, headers); err != nil {
		return err
	}
	fmt.Printf("Output: status=%d\n", s.Response.StatusCode)
	fmt.Printf("Mutation: %s\n", mutationCopy)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iCompleteTheSelfSenderKeySync() error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: the requester device finishes consuming its self sender key sync payload")
	fmt.Println("Action: POST /api/e2ee/self-sender-key-sync/complete")
	if err := s.doE2EEJSONRequest(http.MethodPost, "/api/e2ee/self-sender-key-sync/complete", map[string]any{}, authHeader); err != nil {
		return err
	}
	fmt.Printf("Output: status=%d\n", s.Response.StatusCode)
	fmt.Println("Mutation: self sender key sync may advance to completed")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iFailTheSelfSenderKeySyncWithLastErrorAndRetryable(lastError, retryable string) error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: the current device reports a self sync failure")
	fmt.Printf("Input: last_error=%s retryable=%s\n", lastError, retryable)
	fmt.Println("Action: POST /api/e2ee/self-sender-key-sync/fail")
	payload := map[string]any{
		"last_error": lastError,
		"retryable":  retryable == "true",
	}
	if err := s.doE2EEJSONRequest(http.MethodPost, "/api/e2ee/self-sender-key-sync/fail", payload, authHeader); err != nil {
		return err
	}
	fmt.Printf("Output: status=%d\n", s.Response.StatusCode)
	fmt.Println("Mutation: self sender key sync may reset to waiting or terminal failed")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theSelfSenderKeySyncResponseShouldShowStatus(expected string) error {
	start := time.Now()
	fmt.Println("Given: a self sender key sync response body is available")
	fmt.Printf("Input: expected_status=%s\n", expected)
	fmt.Println("Action: decode self sender key sync response")

	var body selfSenderKeySyncResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	fmt.Printf("Output: exists=%t status=%s requester_current_device=%t provider_current_device=%t\n",
		body.Exists,
		body.Status,
		body.RequesterCurrentDevice,
		body.ProviderCurrentDevice,
	)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if body.Status != expected {
		return fmt.Errorf("expected self sender key sync status %s, got %s", expected, body.Status)
	}
	return nil
}

func (s *steps) theSelfSenderKeySyncRequesterDeviceShouldNowBeBoundAs(deviceIDText, statusText string) error {
	start := time.Now()
	accountID := s.accountBDD.LastSessionAccountID()
	if accountID == 0 {
		return fmt.Errorf("no logged in user available for requester binding assertion")
	}
	deviceID, err := shared.ParseDeviceID(deviceIDText)
	if err != nil {
		return err
	}
	s.deps.SKR.accountRepo.mu.Lock()
	accountData := s.deps.SKR.accountRepo.byID[accountID]
	s.deps.SKR.accountRepo.mu.Unlock()
	if accountData == nil {
		return fmt.Errorf("account not found")
	}
	binding := accountData.GetDevice(deviceID)
	if binding == nil {
		return fmt.Errorf("expected account %d to have device binding %s", accountID, deviceIDText)
	}

	fmt.Println("Given: the requester device binding should reflect the self sync mutation")
	fmt.Printf("Input: account_id=%d device_id=%s expected_status=%s\n", accountID, deviceIDText, statusText)
	fmt.Println("Action: inspect the self sync account binding snapshot")
	fmt.Printf("Output: actual_status=%s\n", binding.Status)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if string(binding.Status) != statusText {
		return fmt.Errorf("expected requester device %s to be %s, got %s", deviceIDText, statusText, binding.Status)
	}
	return nil
}

func (s *steps) doE2EEJSONRequest(method, path string, payload any, authHeader string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, s.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	deviceID := s.accountBDD.LastSessionDeviceID().String()
	req.Header.Set("X-Device-ID", deviceID)

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	s.Response = resp
	s.ResponseBody, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return nil
}
