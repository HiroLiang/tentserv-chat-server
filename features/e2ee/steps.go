package e2ee

import (
	"bytes"
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
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/cucumber/godog"
)

type steps struct {
	*bddsupport.APITestContext
	deps       *Deps
	accountBDD *accountfeatures.Deps
	authHeader string
	start      time.Time
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
	ctx.Step(`^a room member setup exists with room id (\d+), caller member id (\d+), and provider member id (\d+) in the same room with an existing sender key for provider$`, s.roomMemberSetupSameRoomWithKey)
	ctx.Step(`^a room member setup exists with room id (\d+), caller member id (\d+), and provider member id (\d+) where provider is in a different room$`, s.roomMemberSetupDifferentRoom)
	ctx.Step(`^a room member setup exists with room id (\d+), provider member id (\d+) in that room, and the caller has no room membership$`, s.roomMemberSetupCallerNotInRoom)
	ctx.Step(`^I create a sender key request for room (\d+) and provider member (\d+)$`, s.iCreateASenderKeyRequestForRoomAndProviderMember)
	ctx.Step(`^a sender key request row should exist from member (\d+) to provider (\d+)$`, s.aSenderKeyRequestRowShouldExist)
	ctx.Step(`^no sender key request row should exist from member (\d+) to provider (\d+)$`, s.noSenderKeyRequestRowShouldExist)
	ctx.Step(`^pending sender key request count from member (\d+) to provider (\d+) should be (\d+)$`, s.pendingSenderKeyRequestCountFromMemberToProviderShouldBe)
}

func (s *steps) e2eeKeyBootstrapStateIsClean() error {
	s.start = time.Now()
	s.authHeader = ""
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
			"sender_key_public":    "sender-key-public",
			"distribution_message": "distribution-message",
		}, nil
	case "sender-keys":
		return http.MethodGet, "/api/e2ee/sender-keys/1", nil, nil
	case "sender-key-distributions":
		return http.MethodGet, "/api/e2ee/sender-key-distributions/1", nil, nil
	case "sender-key-request":
		return http.MethodPost, "/api/e2ee/sender-key-request", map[string]any{
			"room_id":            1,
			"provider_member_id": 202,
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
	return s.roomMemberSetup(roomID, callerMemberID, providerMemberID, roomID, false)
}

func (s *steps) roomMemberSetupSameRoomWithKey(roomID, callerMemberID, providerMemberID int64) error {
	return s.roomMemberSetup(roomID, callerMemberID, providerMemberID, roomID, true)
}

func (s *steps) roomMemberSetupDifferentRoom(roomID, callerMemberID, providerMemberID int64) error {
	return s.roomMemberSetup(roomID, callerMemberID, providerMemberID, roomID+100, false)
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

func (s *steps) roomMemberSetup(roomID, callerMemberID, providerMemberID, providerRoomID int64, seedProviderKey bool) error {
	s.start = time.Now()
	callerUserID := s.accountBDD.LastSessionUserID()
	if callerUserID == 0 {
		return fmt.Errorf("no logged in user available for room member setup")
	}
	// Use synthetic user IDs derived from member IDs for the provider.
	callerPID := participant.ID(callerMemberID + 1000)
	providerUID := shared.UserID(providerMemberID + 2000)
	providerPID := participant.ID(providerMemberID + 1000)

	s.deps.SKR.SeedMember(callerUserID, callerPID, chatmember.ID(callerMemberID), chatroom.ID(roomID))
	s.deps.SKR.SeedMember(providerUID, providerPID, chatmember.ID(providerMemberID), chatroom.ID(providerRoomID))
	if seedProviderKey {
		s.deps.SKR.SeedProviderKey(chatmember.ID(providerMemberID))
	}
	fmt.Println("Given: room member setup complete")
	fmt.Printf("Input: room_id=%d caller_member=%d provider_member=%d provider_room=%d with_key=%t\n",
		roomID, callerMemberID, providerMemberID, providerRoomID, seedProviderKey)
	fmt.Println("Mutation: SKR participants and members seeded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iCreateASenderKeyRequestForRoomAndProviderMember(roomID, providerMemberID int64) error {
	s.start = time.Now()
	authHeader, err := s.e2eeAuthorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: authenticated user creates a sender key request")
	fmt.Printf("Input: room_id=%d provider_member_id=%d\n", roomID, providerMemberID)
	fmt.Println("Action: POST /api/e2ee/sender-key-request")

	payload := map[string]any{"room_id": roomID, "provider_member_id": providerMemberID}
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
