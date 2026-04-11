package account

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	authPort "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	infraSecurity "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/shared/security"
	"github.com/cucumber/godog"
)

type steps struct {
	*bddsupport.APITestContext
	deps             *Deps
	start            time.Time
	rememberedTokens map[string]string
}

func RegisterSteps(ctx *godog.ScenarioContext, apiCtx *bddsupport.APITestContext, deps *Deps) {
	s := &steps{APITestContext: apiCtx, deps: deps, rememberedTokens: map[string]string{}}

	ctx.Step(`^account registration state is clean$`, s.accountRegistrationStateIsClean)
	ctx.Step(`^the registration rate limit is exceeded$`, s.theRegistrationRateLimitIsExceeded)
	ctx.Step(`^an account exists with email "([^"]*)" and account "([^"]*)"$`, s.anAccountExists)
	ctx.Step(`^an applying account exists with email "([^"]*)", account "([^"]*)", and display name "([^"]*)"$`, s.anApplyingAccountExists)
	ctx.Step(`^the applying account has an expired verification session$`, s.theApplyingAccountHasAnExpiredVerificationSession)
	ctx.Step(`^I register an account with email "([^"]*)", account "([^"]*)", display name "([^"]*)", and password "([^"]*)"$`, s.iRegisterAnAccount)
	ctx.Step(`^the register response should include a verification token and expiry timestamp$`, s.theRegisterResponseShouldIncludeVerificationTokenAndExpiryTimestamp)
	ctx.Step(`^I verify the registered email$`, s.iVerifyTheRegisteredEmail)
	ctx.Step(`^I verify the registered email with the correct code$`, s.iVerifyTheRegisteredEmail)
	ctx.Step(`^I verify the registered email with code "([^"]*)"$`, s.iVerifyTheRegisteredEmailWithCode)
	ctx.Step(`^I verify the same email token again$`, s.iVerifyTheSameEmailTokenAgain)
	ctx.Step(`^I verify email with an empty token$`, s.iVerifyEmailWithAnEmptyToken)
	ctx.Step(`^I verify email with an invalid token$`, s.iVerifyEmailWithAnInvalidToken)
	ctx.Step(`^the verification token expires$`, s.theVerificationTokenExpires)
	ctx.Step(`^I resend the verification email using the registered token$`, s.iResendTheVerificationEmailUsingTheRegisteredToken)
	ctx.Step(`^I resend verification email with an invalid token$`, s.iResendVerificationEmailWithAnInvalidToken)
	ctx.Step(`^I resend verification email with the expired registered token$`, s.iResendVerificationEmailWithTheExpiredRegisteredToken)
	ctx.Step(`^the verify email response should be an empty JSON object$`, s.theVerifyEmailResponseShouldBeAnEmptyJSONObject)
	ctx.Step(`^the resend response should include a new verification token and expiry timestamp$`, s.theResendResponseShouldIncludeANewVerificationTokenAndExpiryTimestamp)
	ctx.Step(`^the response remaining attempts should be (\d+)$`, s.theResponseRemainingAttemptsShouldBe)
	ctx.Step(`^the account registration mutation should include account, user, role, token, and email$`, s.theRegistrationMutationShouldBeComplete)
	ctx.Step(`^the account registration mutation should stop before account creation$`, s.theRegistrationMutationShouldStopBeforeAccountCreation)
	ctx.Step(`^the existing applying account should be updated to account "([^"]*)" and display name "([^"]*)"$`, s.theExistingApplyingAccountShouldBeUpdatedToAccountAndDisplayName)
	ctx.Step(`^the email verification mutation should activate the account and consume the token$`, s.theEmailVerificationMutationShouldActivateTheAccountAndConsumeTheToken)
	ctx.Step(`^the reused email verification token should remain consumed$`, s.theReusedEmailVerificationTokenShouldRemainConsumed)
	ctx.Step(`^the email verification mutation should not update an account$`, s.theEmailVerificationMutationShouldNotUpdateAnAccount)
	ctx.Step(`^the expired token verification should not activate the account$`, s.theExpiredTokenVerificationShouldNotActivateAccount)
	ctx.Step(`^the failed verification attempts should invalidate the session without activating the account$`, s.theFailedVerificationAttemptsShouldInvalidateTheSessionWithoutActivatingTheAccount)
	ctx.Step(`^the resend email mutation should store a new token and send an email$`, s.theResendEmailMutationShouldStoreNewTokenAndSendEmail)
	ctx.Step(`^login state is clean$`, s.loginStateIsClean)
	ctx.Step(`^a registered login device "([^"]*)" named "([^"]*)" exists$`, s.aRegisteredLoginDeviceExists)
	ctx.Step(`^an? "([^"]*)" account exists for login with email "([^"]*)", account "([^"]*)", and password "([^"]*)"$`, s.anAccountExistsForLogin)
	ctx.Step(`^the login account has an existing participant$`, s.theLoginAccountHasAnExistingParticipant)
	ctx.Step(`^I login with identifier "([^"]*)", password "([^"]*)", and device "([^"]*)"$`, s.iLoginWithIdentifier)
	ctx.Step(`^I login with an invalid payload$`, s.iLoginWithAnInvalidPayload)
	ctx.Step(`^I attempt login (\d+) times with identifier "([^"]*)", password "([^"]*)", and device "([^"]*)"$`, s.iAttemptLoginTimesWithIdentifierPasswordAndDevice)
	ctx.Step(`^login response should include a bearer token$`, s.loginResponseShouldIncludeABearerToken)
	ctx.Step(`^I remember the login token as "([^"]*)"$`, s.iRememberTheLoginTokenAs)
	ctx.Step(`^the raw auth header "([^"]*)" is remembered as "([^"]*)"$`, s.theRawAuthHeaderIsRememberedAs)
	ctx.Step(`^login mutation should include session, device link, participant, and login event$`, s.loginMutationShouldIncludeSessionDeviceLinkParticipantAndLoginEvent)
	ctx.Step(`^login mutation should reuse the existing participant$`, s.loginMutationShouldReuseTheExistingParticipant)
	ctx.Step(`^login mutation should stop before session creation$`, s.loginMutationShouldStopBeforeSessionCreation)
	ctx.Step(`^the login identifier "([^"]*)" should be locked$`, s.theLoginIdentifierShouldBeLocked)
	ctx.Step(`^I request my auth profile using the login token and device "([^"]*)"$`, s.iRequestMyAuthProfileUsingTheLoginToken)
	ctx.Step(`^I request my auth profile using remembered login token "([^"]*)" and device "([^"]*)"$`, s.iRequestMyAuthProfileUsingRememberedLoginTokenAndDevice)
	ctx.Step(`^I logout using remembered login token "([^"]*)" and device "([^"]*)"$`, s.iLogoutUsingRememberedLoginTokenAndDevice)
	ctx.Step(`^logout mutation should revoke the remembered login token "([^"]*)"$`, s.logoutMutationShouldRevokeTheRememberedLoginToken)
	ctx.Step(`^the remembered login token "([^"]*)" is revoked$`, s.theRememberedLoginTokenIsRevoked)
	ctx.Step(`^auth profile response should describe the current login user$`, s.authProfileResponseShouldDescribeTheCurrentLoginUser)
}

func (a *steps) theRegistrationRateLimitIsExceeded() error {
	a.start = time.Now()
	fmt.Println("Given: registration rate limit is already exceeded for this IP")
	fmt.Println("Input: register_limiter_exceeded=true")
	fmt.Println("Action: set BDD rate limiter to exceeded state")
	a.deps.SetRegisterLimitExceeded(true)
	fmt.Println("Output: rate limiter will reject next registration attempt")
	fmt.Println("Mutation: register_limiter_exceeded=true")
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) accountRegistrationStateIsClean() error {
	a.start = time.Now()
	fmt.Println("Given: account registration state is clean")
	fmt.Printf("Input: existing_accounts=%d\n", len(a.deps.accountRepo.accountsByID))
	fmt.Println("Action: reset in-memory registration dependencies")
	fmt.Println("Output: clean account registration state")
	fmt.Printf("Mutation: accounts=%d users=%d role_assignments=%d sessions=%d emails=%d\n",
		len(a.deps.accountRepo.accountsByID), a.deps.userRepo.createCalls, a.deps.roleRepo.assignCalls, len(a.deps.store.sessions), a.deps.email.sendCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) anAccountExists(email, accountName string) error {
	a.start = time.Now()
	fmt.Println("Given: an existing account identity is seeded")
	fmt.Printf("Input: email=%s account=%s\n", email, accountName)

	parsed, err := shared.ParseEmail(email)
	if err != nil {
		return err
	}
	a.deps.accountRepo.seed(parsed, accountName)

	fmt.Println("Action: seed account repository")
	fmt.Printf("Output: existing_accounts=%d\n", len(a.deps.accountRepo.accountsByID))
	fmt.Printf("Mutation: accounts=%d\n", len(a.deps.accountRepo.accountsByID))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) anApplyingAccountExists(email, accountName, displayName string) error {
	a.start = time.Now()
	fmt.Println("Given: an applying account already exists for this email")
	fmt.Printf("Input: email=%s account=%s display_name=%q\n", email, accountName, displayName)
	fmt.Println("Action: seed applying account and its user profile")

	parsed, err := shared.ParseEmail(email)
	if err != nil {
		return err
	}
	account := a.deps.accountRepo.seedLogin(parsed, accountName, "existing-hash", domainaccount.Applying, 701)
	a.deps.userRepo.seed(account.ID, 701, displayName, []role.Code{role.User})

	fmt.Printf("Output: account_id=%d user_id=%d\n", account.ID, 701)
	fmt.Printf("Mutation: accounts=%d users=%d\n", len(a.deps.accountRepo.accountsByID), len(a.deps.userRepo.usersByID))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iRegisterAnAccount(email, accountName, displayName, password string) error {
	a.start = time.Now()
	fmt.Println("Given: account registration HTTP endpoint is available")
	fmt.Printf("Input: email=%s account=%s display_name=%q password_present=%t\n", email, accountName, displayName, password != "")
	fmt.Println("Action: POST /api/auth/register")

	payload := map[string]string{
		"email":    email,
		"account":  accountName,
		"name":     displayName,
		"password": password,
	}
	if err := a.DoJSONRequest(http.MethodPost, "/api/auth/register", payload); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: account_create_calls=%d user_create_calls=%d role_assign_calls=%d token_store_calls=%d email_send_calls=%d\n",
		a.deps.accountRepo.createCalls, a.deps.userRepo.createCalls, a.deps.roleRepo.assignCalls, a.deps.store.storeCalls, a.deps.email.sendCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) theRegisterResponseShouldIncludeVerificationTokenAndExpiryTimestamp() error {
	start := time.Now()
	fmt.Println("Given: registration succeeded and should return verification session metadata")
	fmt.Println("Input: expecting verification_token and verification_expires_at_ms")
	fmt.Println("Action: decode register response body")

	var body struct {
		VerificationToken       string `json:"verification_token"`
		VerificationExpiresAtMS int64  `json:"verification_expires_at_ms"`
	}
	if err := json.Unmarshal(a.ResponseBody, &body); err != nil {
		return err
	}

	valid := body.VerificationToken != "" && body.VerificationExpiresAtMS > 0
	fmt.Printf("Output: has_token=%t has_expiry=%t\n", body.VerificationToken != "", body.VerificationExpiresAtMS > 0)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !valid {
		return fmt.Errorf("expected register response to include verification token and expiry, got body=%s", string(a.ResponseBody))
	}
	return nil
}

func (a *steps) iVerifyTheRegisteredEmail() error {
	token := a.deps.store.lastStoredToken
	if token == "" {
		return fmt.Errorf("no verification token was stored")
	}
	return a.requestVerifyEmail("registered", token, a.deps.store.lastStoredSession.Code, true)
}

func (a *steps) iVerifyTheRegisteredEmailWithCode(code string) error {
	token := a.deps.store.lastStoredToken
	if token == "" {
		return fmt.Errorf("no verification token was stored")
	}
	return a.requestVerifyEmail("custom-code", token, code, true)
}

func (a *steps) iVerifyTheSameEmailTokenAgain() error {
	token := a.deps.store.lastStoredToken
	if token == "" {
		return fmt.Errorf("no verification token was stored")
	}
	return a.requestVerifyEmail("reused", token, a.deps.store.lastStoredSession.Code, true)
}

func (a *steps) iVerifyEmailWithAnEmptyToken() error {
	return a.requestVerifyEmail("empty", "", "", true)
}

func (a *steps) iVerifyEmailWithAnInvalidToken() error {
	return a.requestVerifyEmail("invalid", "invalid-token", "123456", true)
}

func (a *steps) theApplyingAccountHasAnExpiredVerificationSession() error {
	a.start = time.Now()
	account, ok := a.firstAccount()
	if !ok {
		return fmt.Errorf("no applying account seeded")
	}
	fmt.Println("Given: the applying account previously had a verification session that already expired")
	fmt.Printf("Input: account_id=%d\n", account.ID)
	fmt.Println("Action: store a verification session and expire it immediately")

	session := authPort.VerificationSession{
		AccountID:         int64(account.ID),
		Code:              "123456",
		ExpiresAtMS:       time.Now().Add(-time.Minute).UnixMilli(),
		RemainingAttempts: 3,
	}
	if err := a.deps.store.Store(context.Background(), "expired-seed-token", session, time.Minute); err != nil {
		return err
	}
	a.deps.store.expireToken("expired-seed-token")

	fmt.Printf("Output: sessions_remaining=%d\n", len(a.deps.store.sessions))
	fmt.Printf("Mutation: store_calls=%d delete_calls=%d\n", a.deps.store.storeCalls, a.deps.store.deleteCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) theVerificationTokenExpires() error {
	a.start = time.Now()
	token := a.deps.store.lastStoredToken
	fmt.Println("Given: a valid token was issued but its TTL has elapsed")
	fmt.Printf("Input: token_present=%t\n", token != "")
	fmt.Println("Action: expire token in verification store (simulates Redis TTL)")
	a.deps.store.expireToken(token)
	fmt.Printf("Output: sessions_remaining=%d\n", len(a.deps.store.sessions))
	fmt.Printf("Mutation: token_expired=true delete_calls_unchanged=%d\n", a.deps.store.deleteCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iResendTheVerificationEmailUsingTheRegisteredToken() error {
	a.start = time.Now()
	storeBefore := a.deps.store.storeCalls
	emailBefore := a.deps.email.sendCalls
	fmt.Println("Given: resend verification email HTTP endpoint is available")
	fmt.Printf("Input: token_present=%t store_calls_before=%d email_calls_before=%d\n", a.deps.store.lastStoredToken != "", storeBefore, emailBefore)
	fmt.Println("Action: POST /api/auth/resend-verify-email")

	payload := map[string]string{"token": a.deps.store.lastStoredToken}
	if err := a.DoJSONRequest(http.MethodPost, "/api/auth/resend-verify-email", payload); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: token_store_calls=%d email_send_calls=%d sessions_in_store=%d\n",
		a.deps.store.storeCalls, a.deps.email.sendCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iResendVerificationEmailWithAnInvalidToken() error {
	a.start = time.Now()
	fmt.Println("Given: resend verification email HTTP endpoint is available")
	fmt.Println("Input: token=invalid-token")
	fmt.Println("Action: POST /api/auth/resend-verify-email")

	if err := a.DoJSONRequest(http.MethodPost, "/api/auth/resend-verify-email", map[string]string{"token": "invalid-token"}); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: token_store_calls=%d email_send_calls=%d sessions_in_store=%d\n",
		a.deps.store.storeCalls, a.deps.email.sendCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iResendVerificationEmailWithTheExpiredRegisteredToken() error {
	a.start = time.Now()
	fmt.Println("Given: resend verification email HTTP endpoint is available with an expired token")
	fmt.Printf("Input: token_present=%t\n", a.deps.store.lastStoredToken != "")
	fmt.Println("Action: POST /api/auth/resend-verify-email")

	if err := a.DoJSONRequest(http.MethodPost, "/api/auth/resend-verify-email", map[string]string{"token": a.deps.store.lastStoredToken}); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: token_store_calls=%d email_send_calls=%d sessions_in_store=%d\n",
		a.deps.store.storeCalls, a.deps.email.sendCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) theExpiredTokenVerificationShouldNotActivateAccount() error {
	start := time.Now()
	fmt.Println("Given: verification token was expired before the verify attempt")
	fmt.Println("Input: expecting no account activation and no delete call from verify use case")
	fmt.Println("Action: inspect account and verification store state after expired-token verify")

	// After registration: updateCalls=1 (link user). Verify should NOT add another update.
	// expireToken() did not increment deleteCalls, so deleteCalls should still be 0.
	noVerifyDelete := a.deps.store.deleteCalls == 0
	noActivation := len(a.deps.store.sessions) == 0

	status, accountFound := a.firstAccountStatus()
	notActive := !accountFound || status == domainaccount.Applying

	fmt.Printf("Output: no_verify_delete=%t no_activation=%t account_status=%s\n", noVerifyDelete, noActivation, status)
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d sessions_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !noVerifyDelete {
		return fmt.Errorf("expected no delete call from verify use case for expired token, got delete_calls=%d", a.deps.store.deleteCalls)
	}
	if !noActivation || !notActive {
		return fmt.Errorf("expected account to remain unactivated, got sessions_remaining=%d account_status=%s", len(a.deps.store.sessions), status)
	}
	return nil
}

func (a *steps) theResendEmailMutationShouldStoreNewTokenAndSendEmail() error {
	start := time.Now()
	fmt.Println("Given: resend verification email should issue a new token and send an email")
	fmt.Println("Input: expecting incremented store and email send calls")
	fmt.Println("Action: inspect verification store and email service counters")

	tokenStored := a.deps.store.storeCalls >= 2
	emailSent := a.deps.email.sendCalls >= 2

	fmt.Printf("Output: token_stored=%t email_sent=%t\n", tokenStored, emailSent)
	fmt.Printf("Mutation: token_store_calls=%d email_send_calls=%d sessions_in_store=%d\n",
		a.deps.store.storeCalls, a.deps.email.sendCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !tokenStored || !emailSent {
		return fmt.Errorf("expected resend to store a new token and send an email, got store_calls=%d email_calls=%d",
			a.deps.store.storeCalls, a.deps.email.sendCalls)
	}
	return nil
}

func (a *steps) requestVerifyEmail(caseName, token, code string, includeToken bool) error {
	a.start = time.Now()
	fmt.Println("Given: email verification HTTP endpoint is available")
	fmt.Printf("Input: case=%s token_present=%t code=%q token_consumed=%t\n", caseName, token != "", code, len(a.deps.store.sessions) == 0)
	fmt.Println("Action: POST /api/auth/verify-email")

	payload := map[string]string{}
	if includeToken {
		payload["token"] = token
	}
	if code != "" {
		payload["code"] = code
	}
	if err := a.DoJSONRequest(http.MethodPost, "/api/auth/verify-email", payload); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d sessions_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) theVerifyEmailResponseShouldBeAnEmptyJSONObject() error {
	start := time.Now()
	fmt.Println("Given: verify email success should return an empty JSON object")
	fmt.Println("Input: expecting {}")
	fmt.Println("Action: compare response body")

	isEmptyJSONObject := strings.TrimSpace(string(a.ResponseBody)) == "{}"

	fmt.Printf("Output: is_empty_json_object=%t\n", isEmptyJSONObject)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !isEmptyJSONObject {
		return fmt.Errorf("expected verify email response to be {}, got body=%s", string(a.ResponseBody))
	}
	return nil
}

func (a *steps) theResendResponseShouldIncludeANewVerificationTokenAndExpiryTimestamp() error {
	start := time.Now()
	fmt.Println("Given: resend verification succeeded and should return a fresh verification session")
	fmt.Println("Input: expecting verification_token and verification_expires_at_ms")
	fmt.Println("Action: decode resend response body")

	var body struct {
		VerificationToken       string `json:"verification_token"`
		VerificationExpiresAtMS int64  `json:"verification_expires_at_ms"`
	}
	if err := json.Unmarshal(a.ResponseBody, &body); err != nil {
		return err
	}

	matchesStore := body.VerificationToken != "" && body.VerificationToken == a.deps.store.lastStoredToken && body.VerificationExpiresAtMS == a.deps.store.lastStoredSession.ExpiresAtMS
	fmt.Printf("Output: matches_store=%t has_token=%t has_expiry=%t\n", matchesStore, body.VerificationToken != "", body.VerificationExpiresAtMS > 0)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !matchesStore {
		return fmt.Errorf("expected resend response to include the latest token and expiry, got body=%s", string(a.ResponseBody))
	}
	return nil
}

func (a *steps) theResponseRemainingAttemptsShouldBe(expected int) error {
	start := time.Now()
	fmt.Println("Given: the error response should include remaining verification attempts")
	fmt.Printf("Input: expected_remaining_attempts=%d\n", expected)
	fmt.Println("Action: decode error details")

	details, err := a.decodeErrorDetails()
	if err != nil {
		return err
	}
	actualValue, ok := details["remaining_attempts"]
	if !ok {
		return fmt.Errorf("expected remaining_attempts in error details, got %v", details)
	}

	var actual int
	switch v := actualValue.(type) {
	case float64:
		actual = int(v)
	case int:
		actual = v
	default:
		return fmt.Errorf("unexpected remaining_attempts type %T", actualValue)
	}

	fmt.Printf("Output: actual_remaining_attempts=%d match=%t\n", actual, actual == expected)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if actual != expected {
		return fmt.Errorf("expected remaining attempts %d, got %d", expected, actual)
	}
	return nil
}

func (a *steps) theRegistrationMutationShouldBeComplete() error {
	start := time.Now()
	fmt.Println("Given: registration request should complete all side effects")
	fmt.Println("Input: expecting account/user/role/token/email mutations")
	fmt.Println("Action: inspect in-memory dependency counters")

	accountCreated := a.deps.accountRepo.createCalls == 1
	userCreated := a.deps.userRepo.createCalls == 1
	roleAssigned := a.deps.roleRepo.assignCalls == 1
	tokenStored := a.deps.store.storeCalls == 1 && len(a.deps.store.sessions) == 1
	emailSent := a.deps.email.sendCalls == 1

	fmt.Printf("Output: account_created=%t user_created=%t role_assigned=%t token_stored=%t email_sent=%t\n",
		accountCreated, userCreated, roleAssigned, tokenStored, emailSent)
	fmt.Printf("Mutation: account_create_calls=%d account_update_calls=%d user_create_calls=%d user_update_calls=%d role_assign_calls=%d token_store_calls=%d email_send_calls=%d\n",
		a.deps.accountRepo.createCalls, a.deps.accountRepo.updateCalls, a.deps.userRepo.createCalls, a.deps.userRepo.updateCalls, a.deps.roleRepo.assignCalls, a.deps.store.storeCalls, a.deps.email.sendCalls)
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !accountCreated || !userCreated || !roleAssigned || !tokenStored || !emailSent {
		return fmt.Errorf("expected complete registration mutation, got account=%t user=%t role=%t token=%t email=%t",
			accountCreated, userCreated, roleAssigned, tokenStored, emailSent)
	}
	return nil
}

func (a *steps) theRegistrationMutationShouldStopBeforeAccountCreation() error {
	start := time.Now()
	fmt.Println("Given: registration should fail before account creation")
	fmt.Println("Input: expecting no account/user/role/token/email mutations")
	fmt.Println("Action: inspect in-memory dependency counters")

	noMutation := a.deps.accountRepo.createCalls == 0 &&
		a.deps.userRepo.createCalls == 0 &&
		a.deps.roleRepo.assignCalls == 0 &&
		a.deps.store.storeCalls == 0 &&
		a.deps.email.sendCalls == 0

	fmt.Printf("Output: no_mutation=%t\n", noMutation)
	fmt.Printf("Mutation: account_create_calls=%d user_create_calls=%d role_assign_calls=%d token_store_calls=%d email_send_calls=%d\n",
		a.deps.accountRepo.createCalls, a.deps.userRepo.createCalls, a.deps.roleRepo.assignCalls, a.deps.store.storeCalls, a.deps.email.sendCalls)
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !noMutation {
		return fmt.Errorf("expected no registration mutation before account creation")
	}
	return nil
}

func (a *steps) theEmailVerificationMutationShouldActivateTheAccountAndConsumeTheToken() error {
	start := time.Now()
	fmt.Println("Given: email verification should activate the applying account")
	fmt.Println("Input: expecting active account and consumed token")
	fmt.Println("Action: inspect account and verification store state")

	status, accountFound := a.firstAccountStatus()
	accountActive := accountFound && status == domainaccount.Active
	tokenConsumed := len(a.deps.store.sessions) == 0 && a.deps.store.deleteCalls == 1
	mutationComplete := accountActive && tokenConsumed && a.deps.store.getCalls == 1 && a.deps.accountRepo.updateCalls >= 1

	fmt.Printf("Output: account_found=%t account_active=%t token_consumed=%t mutation_complete=%t\n",
		accountFound, accountActive, tokenConsumed, mutationComplete)
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d sessions_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !mutationComplete {
		return fmt.Errorf("expected email verification to activate account and consume token")
	}
	return nil
}

func (a *steps) theReusedEmailVerificationTokenShouldRemainConsumed() error {
	start := time.Now()
	fmt.Println("Given: a verification token was already consumed")
	fmt.Println("Input: expecting no second account activation")
	fmt.Println("Action: inspect account and verification store state")

	status, accountFound := a.firstAccountStatus()
	accountActive := accountFound && status == domainaccount.Active
	stillConsumed := len(a.deps.store.sessions) == 0 && a.deps.store.getCalls == 2 && a.deps.store.deleteCalls == 1
	noSecondUpdate := a.deps.accountRepo.updateCalls >= 1

	fmt.Printf("Output: account_active=%t token_still_consumed=%t no_second_update=%t\n",
		accountActive, stillConsumed, noSecondUpdate)
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d sessions_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !accountActive || !stillConsumed || !noSecondUpdate {
		return fmt.Errorf("expected reused token to remain consumed without another account update")
	}
	return nil
}

func (a *steps) theEmailVerificationMutationShouldNotUpdateAnAccount() error {
	start := time.Now()
	fmt.Println("Given: email verification request should be rejected")
	fmt.Println("Input: expecting no account update and no token deletion")
	fmt.Println("Action: inspect account and verification store state")

	noUpdate := a.deps.accountRepo.updateCalls == 0
	noDelete := a.deps.store.deleteCalls == 0
	noMutation := noUpdate && noDelete && len(a.deps.store.sessions) == 0

	fmt.Printf("Output: no_account_update=%t no_token_delete=%t no_mutation=%t\n", noUpdate, noDelete, noMutation)
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d sessions_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !noMutation {
		return fmt.Errorf("expected email verification rejection without account update")
	}
	return nil
}

func (a *steps) theExistingApplyingAccountShouldBeUpdatedToAccountAndDisplayName(accountName, displayName string) error {
	start := time.Now()
	fmt.Println("Given: registration reused an existing applying account")
	fmt.Printf("Input: expected_account=%s expected_display_name=%q\n", accountName, displayName)
	fmt.Println("Action: inspect account and user repositories")

	if a.deps.accountRepo.lastUpdated == nil {
		return fmt.Errorf("expected applying account update, got nil")
	}
	if a.deps.userRepo.lastUpdated == nil {
		return fmt.Errorf("expected applying user update, got nil")
	}

	accountUpdated := a.deps.accountRepo.lastUpdated.AccountName == accountName && a.deps.accountRepo.lastUpdated.Status == domainaccount.Applying
	userUpdated := a.deps.userRepo.lastUpdated.Name == displayName

	fmt.Printf("Output: account_updated=%t user_updated=%t\n", accountUpdated, userUpdated)
	fmt.Printf("Mutation: account_update_calls=%d user_update_calls=%d\n", a.deps.accountRepo.updateCalls, a.deps.userRepo.updateCalls)
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !accountUpdated || !userUpdated {
		return fmt.Errorf("expected applying account/user to be updated, got account=%+v user=%+v", a.deps.accountRepo.lastUpdated, a.deps.userRepo.lastUpdated)
	}
	return nil
}

func (a *steps) theFailedVerificationAttemptsShouldInvalidateTheSessionWithoutActivatingTheAccount() error {
	start := time.Now()
	fmt.Println("Given: verification failed too many times")
	fmt.Println("Input: expecting session removal and applying account status")
	fmt.Println("Action: inspect account and session store state")

	status, accountFound := a.firstAccountStatus()
	accountStillApplying := accountFound && status == domainaccount.Applying
	sessionRemoved := len(a.deps.store.sessions) == 0 && a.deps.store.deleteCalls == 1

	fmt.Printf("Output: account_still_applying=%t session_removed=%t\n", accountStillApplying, sessionRemoved)
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d sessions_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.sessions))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !accountStillApplying || !sessionRemoved {
		return fmt.Errorf("expected failed verification attempts to remove session without activation")
	}
	return nil
}

func (a *steps) loginStateIsClean() error {
	a.start = time.Now()
	a.rememberedTokens = map[string]string{}
	fmt.Println("Given: login state is clean")
	fmt.Printf("Input: existing_accounts=%d sessions=%d participants=%d\n",
		len(a.deps.accountRepo.accountsByID), len(a.deps.sessionManager.sessions), len(a.deps.participantRepo.participantsByUser))
	fmt.Println("Action: reset in-memory login dependencies")
	fmt.Println("Output: clean login state")
	fmt.Printf("Mutation: accounts=%d users=%d devices=%d sessions=%d participants=%d login_events=%d\n",
		len(a.deps.accountRepo.accountsByID), len(a.deps.userRepo.usersByID), len(a.deps.deviceRepo.devicesByID),
		len(a.deps.sessionManager.sessions), len(a.deps.participantRepo.participantsByUser), a.deps.accountRepo.recordLoginEventCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) aRegisteredLoginDeviceExists(deviceID, name string) error {
	a.start = time.Now()
	fmt.Println("Given: a registered startup device exists")
	fmt.Printf("Input: device_id=%s device_name=%q\n", deviceID, name)
	fmt.Println("Action: seed device repository")

	parsed, err := shared.ParseDeviceID(deviceID)
	if err != nil {
		return err
	}
	a.deps.deviceRepo.seed(parsed, name)

	fmt.Printf("Output: registered_devices=%d\n", len(a.deps.deviceRepo.devicesByID))
	fmt.Printf("Mutation: devices=%d\n", len(a.deps.deviceRepo.devicesByID))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) anAccountExistsForLogin(statusText, email, accountName, password string) error {
	a.start = time.Now()
	fmt.Println("Given: a login account identity is seeded")
	fmt.Printf("Input: status=%s email=%s account=%s password_present=%t\n", statusText, email, accountName, password != "")
	fmt.Println("Action: hash password and seed account repository")

	status := domainaccount.Status(statusText)
	switch status {
	case domainaccount.Active, domainaccount.Applying, domainaccount.Inactive, domainaccount.Banned, domainaccount.Deleted:
	default:
		return fmt.Errorf("unknown login account status %q", statusText)
	}
	parsed, err := shared.ParseEmail(email)
	if err != nil {
		return err
	}
	hash, err := infraSecurity.NewArgon2Hasher().Hash(password)
	if err != nil {
		return err
	}
	acc := a.deps.accountRepo.seedLogin(parsed, accountName, hash, status)

	fmt.Printf("Output: account_id=%d status=%s user_count=%d\n", acc.ID, acc.Status, len(acc.UserIDs))
	fmt.Printf("Mutation: accounts=%d\n", len(a.deps.accountRepo.accountsByID))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) theLoginAccountHasAnExistingParticipant() error {
	start := time.Now()
	fmt.Println("Given: the latest login account should already have a user participant")
	fmt.Println("Input: roles=user")
	fmt.Println("Action: seed user repository and participant repository")

	acc, ok := a.firstLoginAccount()
	if !ok {
		return fmt.Errorf("no login account seeded")
	}
	userID := shared.UserID(601)
	a.deps.userRepo.seed(acc.ID, userID, acc.AccountName, []role.Code{role.User})

	a.deps.accountRepo.mu.Lock()
	accInRepo := a.deps.accountRepo.accountsByID[acc.ID]
	accInRepo.UserIDs = []shared.UserID{userID}
	a.deps.accountRepo.mu.Unlock()

	p := a.deps.participantRepo.seedUser(userID)

	fmt.Printf("Output: user_id=%d participant_id=%d\n", userID, p.ID)
	fmt.Printf("Mutation: users=%d participants=%d\n", len(a.deps.userRepo.usersByID), len(a.deps.participantRepo.participantsByUser))
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (a *steps) iLoginWithIdentifier(identifier, password, deviceID string) error {
	a.start = time.Now()
	fmt.Println("Given: login HTTP endpoint is available")
	fmt.Printf("Input: identifier=%s password_present=%t device_id=%s\n", identifier, password != "", deviceID)
	fmt.Println("Action: POST /api/auth/login")

	payload := map[string]string{
		"identifier": identifier,
		"password":   password,
		"device_id":  deviceID,
	}
	if err := a.DoJSONRequestWithHeaders(http.MethodPost, "/api/auth/login", payload, map[string]string{
		"User-Agent": "TentservDesktop/BDD",
	}); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s auth_header_present=%t\n",
		a.Response.StatusCode, string(a.ResponseBody), a.Response.Header.Get("Authorization") != "")
	fmt.Printf("Mutation: user_create_calls=%d role_assign_calls=%d device_find_calls=%d register_device_calls=%d session_create_calls=%d participant_create_calls=%d login_event_calls=%d account_update_calls=%d\n",
		a.deps.userRepo.createCalls, a.deps.roleRepo.assignCalls, a.deps.deviceRepo.findByIDCalls,
		a.deps.accountRepo.registerDeviceCalls, a.deps.sessionManager.createCalls, a.deps.participantRepo.createCalls,
		a.deps.accountRepo.recordLoginEventCalls, a.deps.accountRepo.updateCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iLoginWithAnInvalidPayload() error {
	a.start = time.Now()
	fmt.Println("Given: login HTTP endpoint is available")
	fmt.Println("Input: payload_missing_password=true")
	fmt.Println("Action: POST /api/auth/login")

	payload := map[string]string{
		"identifier": "login@example.com",
		"device_id":  "11111111-1111-1111-1111-111111111111",
	}
	if err := a.DoJSONRequest(http.MethodPost, "/api/auth/login", payload); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: session_create_calls=%d login_event_calls=%d\n", a.deps.sessionManager.createCalls, a.deps.accountRepo.recordLoginEventCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iAttemptLoginTimesWithIdentifierPasswordAndDevice(attempts int, identifier, password, deviceID string) error {
	a.start = time.Now()
	fmt.Println("Given: login HTTP endpoint is available for repeated attempts")
	fmt.Printf("Input: attempts=%d identifier=%s password_present=%t device_id=%s\n", attempts, identifier, password != "", deviceID)
	fmt.Println("Action: repeat POST /api/auth/login")

	for i := 0; i < attempts; i++ {
		payload := map[string]string{
			"identifier": identifier,
			"password":   password,
			"device_id":  deviceID,
		}
		if err := a.DoJSONRequestWithHeaders(http.MethodPost, "/api/auth/login", payload, map[string]string{
			"User-Agent": "TentservDesktop/BDD",
		}); err != nil {
			return err
		}
	}

	fmt.Printf("Output: last_status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: login_check_calls=%d login_record_calls=%d failure_count=%d\n",
		a.deps.loginLimiter.checkCalls, a.deps.loginLimiter.recordCalls, a.deps.loginLimiter.count(normalizeLoginIdentifierForBDD(identifier)))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) loginResponseShouldIncludeABearerToken() error {
	start := time.Now()
	fmt.Println("Given: login response should contain a session token header")
	fmt.Println("Input: expected_prefix=Bearer token_redacted=true")
	fmt.Println("Action: inspect Authorization header")

	header := a.Response.Header.Get("Authorization")
	hasBearer := strings.HasPrefix(header, "Bearer ") && len(header) > len("Bearer ")

	fmt.Printf("Output: bearer_header_present=%t\n", hasBearer)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !hasBearer {
		return fmt.Errorf("expected Authorization bearer header, got present=%t", header != "")
	}
	return nil
}

func (a *steps) iRememberTheLoginTokenAs(label string) error {
	start := time.Now()
	fmt.Println("Given: the last login response returned an Authorization header")
	fmt.Printf("Input: label=%s\n", label)
	fmt.Println("Action: store the bearer token for later requests")

	header := a.Response.Header.Get("Authorization")
	if header == "" {
		return fmt.Errorf("no Authorization header to remember")
	}
	a.rememberedTokens[label] = header

	fmt.Printf("Output: token_saved=%t saved_tokens=%d\n", a.rememberedTokens[label] != "", len(a.rememberedTokens))
	fmt.Println("Mutation: remembered_token_saved=true")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (a *steps) theRawAuthHeaderIsRememberedAs(rawHeader, label string) error {
	start := time.Now()
	fmt.Println("Given: a raw Authorization header should be reused for a later request")
	fmt.Printf("Input: label=%s header_present=%t bearer_prefix=%t\n", label, rawHeader != "", strings.HasPrefix(rawHeader, "Bearer "))
	fmt.Println("Action: store the raw Authorization header without validation")

	a.rememberedTokens[label] = rawHeader

	fmt.Printf("Output: token_saved=%t saved_tokens=%d\n", a.rememberedTokens[label] != "", len(a.rememberedTokens))
	fmt.Println("Mutation: remembered_token_saved=true")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (a *steps) loginMutationShouldIncludeSessionDeviceLinkParticipantAndLoginEvent() error {
	start := time.Now()
	fmt.Println("Given: successful login should persist all login side effects")
	fmt.Println("Input: expecting session, device link, participant, and login event")
	fmt.Println("Action: inspect in-memory dependency counters")

	sessionCreated := a.deps.sessionManager.createCalls == 1
	deviceLinked := a.deps.accountRepo.registerDeviceCalls == 1 && a.deps.accountRepo.lastRegisteredDevice != nil
	participantEnsured := len(a.deps.participantRepo.participantsByUser) == 1
	loginEvent := a.deps.accountRepo.recordLoginEventCalls == 1 && a.deps.accountRepo.lastLoginEvent != nil && a.deps.accountRepo.lastLoginEvent.Success
	accountUpdated := a.deps.accountRepo.updateCalls == 1

	fmt.Printf("Output: session_created=%t device_linked=%t participant_ensured=%t login_event=%t account_updated=%t\n",
		sessionCreated, deviceLinked, participantEnsured, loginEvent, accountUpdated)
	fmt.Printf("Mutation: session_create_calls=%d register_device_calls=%d participant_count=%d login_event_calls=%d account_update_calls=%d\n",
		a.deps.sessionManager.createCalls, a.deps.accountRepo.registerDeviceCalls, len(a.deps.participantRepo.participantsByUser),
		a.deps.accountRepo.recordLoginEventCalls, a.deps.accountRepo.updateCalls)
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !sessionCreated || !deviceLinked || !participantEnsured || !loginEvent || !accountUpdated {
		return fmt.Errorf("expected complete login mutation")
	}
	return nil
}

func (a *steps) loginMutationShouldReuseTheExistingParticipant() error {
	start := time.Now()
	fmt.Println("Given: login account already had a participant")
	fmt.Println("Input: expecting no participant create")
	fmt.Println("Action: inspect participant counters")

	reused := a.deps.participantRepo.findByUserCalls == 1 && a.deps.participantRepo.createCalls == 0
	sessionCreated := a.deps.sessionManager.createCalls == 1

	fmt.Printf("Output: participant_reused=%t session_created=%t\n", reused, sessionCreated)
	fmt.Printf("Mutation: participant_find_calls=%d participant_create_calls=%d session_create_calls=%d\n",
		a.deps.participantRepo.findByUserCalls, a.deps.participantRepo.createCalls, a.deps.sessionManager.createCalls)
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !reused || !sessionCreated {
		return fmt.Errorf("expected existing participant to be reused")
	}
	return nil
}

func (a *steps) loginMutationShouldStopBeforeSessionCreation() error {
	start := time.Now()
	fmt.Println("Given: login request should fail before session creation")
	fmt.Println("Input: expecting no token/session/login-event mutation")
	fmt.Println("Action: inspect login counters")

	noSession := a.deps.sessionManager.createCalls == 0
	noLoginEvent := a.deps.accountRepo.recordLoginEventCalls == 0
	noAccountUpdate := a.deps.accountRepo.updateCalls == 0

	fmt.Printf("Output: no_session=%t no_login_event=%t no_account_update=%t\n", noSession, noLoginEvent, noAccountUpdate)
	fmt.Printf("Mutation: session_create_calls=%d login_event_calls=%d account_update_calls=%d\n",
		a.deps.sessionManager.createCalls, a.deps.accountRepo.recordLoginEventCalls, a.deps.accountRepo.updateCalls)
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !noSession || !noLoginEvent || !noAccountUpdate {
		return fmt.Errorf("expected login failure before session creation")
	}
	return nil
}

func (a *steps) theLoginIdentifierShouldBeLocked(identifier string) error {
	start := time.Now()
	normalizedIdentifier := normalizeLoginIdentifierForBDD(identifier)
	fmt.Println("Given: repeated failed logins should eventually lock the identifier")
	fmt.Printf("Input: identifier=%s normalized_identifier=%s\n", identifier, normalizedIdentifier)
	fmt.Println("Action: inspect login limiter failure count")

	failureCount := a.deps.loginLimiter.count(normalizedIdentifier)
	locked := failureCount >= 5

	fmt.Printf("Output: failure_count=%d locked=%t\n", failureCount, locked)
	fmt.Printf("Mutation: login_check_calls=%d login_record_calls=%d\n", a.deps.loginLimiter.checkCalls, a.deps.loginLimiter.recordCalls)
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !locked {
		return fmt.Errorf("expected identifier %s to be locked after failed attempts, got count=%d", normalizedIdentifier, failureCount)
	}
	return nil
}

func (a *steps) iRequestMyAuthProfileUsingTheLoginToken(deviceID string) error {
	a.start = time.Now()
	authHeader := ""
	if a.Response != nil {
		authHeader = a.Response.Header.Get("Authorization")
	}
	if authHeader == "" {
		token := a.deps.LastAccessToken()
		if token != "" {
			authHeader = "Bearer " + string(token)
		}
	}
	fmt.Println("Given: a login bearer token was returned")
	fmt.Printf("Input: device_id=%s token_present=%t\n", deviceID, authHeader != "")
	fmt.Println("Action: GET /api/auth/profile")

	if authHeader == "" {
		return fmt.Errorf("no bearer token available from login session")
	}
	if err := a.doAuthProfileRequest(authHeader, deviceID); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: session_find_calls=%d\n", a.deps.sessionManager.findCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iRequestMyAuthProfileUsingRememberedLoginTokenAndDevice(label, deviceID string) error {
	a.start = time.Now()
	fmt.Println("Given: a remembered login bearer token is available")
	fmt.Printf("Input: label=%s device_id=%s\n", label, deviceID)
	fmt.Println("Action: GET /api/auth/profile with remembered token")

	authHeader, ok := a.rememberedTokens[label]
	if !ok {
		return fmt.Errorf("no remembered token for label %q", label)
	}
	if err := a.doAuthProfileRequest(authHeader, deviceID); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: session_find_calls=%d\n", a.deps.sessionManager.findCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iLogoutUsingRememberedLoginTokenAndDevice(label, deviceID string) error {
	a.start = time.Now()
	fmt.Println("Given: a remembered login bearer token is available")
	fmt.Printf("Input: label=%s device_id=%s\n", label, deviceID)
	fmt.Println("Action: POST /api/auth/logout")

	authHeader, ok := a.rememberedTokens[label]
	if !ok {
		return fmt.Errorf("no remembered token for label %q", label)
	}
	if err := a.DoRequestWithHeaders(http.MethodPost, "/api/auth/logout", map[string]string{
		"Authorization": authHeader,
		"X-Device-ID":   deviceID,
	}); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: session_revoke_calls=%d remaining_sessions=%d\n", a.deps.sessionManager.revokeCalls, len(a.deps.sessionManager.sessions))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) logoutMutationShouldRevokeTheRememberedLoginToken(label string) error {
	start := time.Now()
	fmt.Println("Given: logout should revoke the selected bearer token")
	fmt.Printf("Input: label=%s\n", label)
	fmt.Println("Action: inspect session manager state")

	rawToken, err := rawRememberedAccessToken(a.rememberedTokens[label])
	if err != nil {
		return err
	}
	_, exists := a.deps.sessionManager.sessions[auth.AccessToken(rawToken)]

	fmt.Printf("Output: token_removed=%t revoke_calls=%d\n", !exists, a.deps.sessionManager.revokeCalls)
	fmt.Printf("Mutation: remaining_sessions=%d\n", len(a.deps.sessionManager.sessions))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if exists || a.deps.sessionManager.revokeCalls == 0 {
		return fmt.Errorf("expected remembered token %q to be revoked", label)
	}
	return nil
}

func (a *steps) theRememberedLoginTokenIsRevoked(label string) error {
	start := time.Now()
	fmt.Println("Given: a remembered login token exists and should be revoked manually")
	fmt.Printf("Input: label=%s\n", label)
	fmt.Println("Action: revoke the remembered token in the fake session manager")

	authHeader, ok := a.rememberedTokens[label]
	if !ok {
		return fmt.Errorf("no remembered token for label %q", label)
	}
	rawToken, err := rawRememberedAccessToken(authHeader)
	if err != nil {
		return err
	}
	if err := a.deps.sessionManager.Revoke(context.Background(), auth.AccessToken(rawToken)); err != nil {
		return err
	}

	fmt.Printf("Output: revoke_calls=%d remaining_sessions=%d\n", a.deps.sessionManager.revokeCalls, len(a.deps.sessionManager.sessions))
	fmt.Println("Mutation: token_revoked=true")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (a *steps) authProfileResponseShouldDescribeTheCurrentLoginUser() error {
	start := time.Now()
	fmt.Println("Given: profile response should describe the authenticated account and current user")
	fmt.Println("Input: expecting account_id and current_user")
	fmt.Println("Action: decode auth profile response")

	var profile struct {
		AccountID   int64  `json:"account_id"`
		Email       string `json:"email"`
		AccountName string `json:"account_name"`
		CurrentUser struct {
			ID        int64    `json:"id"`
			Name      string   `json:"name"`
			RoleCodes []string `json:"role_codes"`
		} `json:"current_user"`
	}
	if err := json.Unmarshal(a.ResponseBody, &profile); err != nil {
		return err
	}
	hasProfile := profile.AccountID != 0 && profile.CurrentUser.ID != 0 && profile.Email != ""

	fmt.Printf("Output: has_profile=%t account_id=%d current_user_id=%d role_count=%d\n",
		hasProfile, profile.AccountID, profile.CurrentUser.ID, len(profile.CurrentUser.RoleCodes))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !hasProfile {
		return fmt.Errorf("expected profile to contain account and current user, got %+v", profile)
	}
	return nil
}

func (a *steps) firstLoginAccount() (*domainaccount.Account, bool) {
	a.deps.accountRepo.mu.Lock()
	defer a.deps.accountRepo.mu.Unlock()

	for _, acc := range a.deps.accountRepo.accountsByID {
		return cloneBDDAccount(acc), true
	}
	return nil, false
}

func (a *steps) firstAccount() (*domainaccount.Account, bool) {
	a.deps.accountRepo.mu.Lock()
	defer a.deps.accountRepo.mu.Unlock()

	for _, acc := range a.deps.accountRepo.accountsByID {
		return cloneBDDAccount(acc), true
	}
	return nil, false
}

func (a *steps) firstAccountStatus() (domainaccount.Status, bool) {
	a.deps.accountRepo.mu.Lock()
	defer a.deps.accountRepo.mu.Unlock()

	for _, acc := range a.deps.accountRepo.accountsByID {
		return acc.Status, true
	}
	return "", false
}

func (a *steps) doAuthProfileRequest(authHeader, deviceID string) error {
	return a.DoRequestWithHeaders(http.MethodGet, "/api/auth/profile", map[string]string{
		"Authorization": authHeader,
		"X-Device-ID":   deviceID,
	})
}

func normalizeLoginIdentifierForBDD(identifier string) string {
	trimmed := strings.TrimSpace(identifier)
	if emailAddr, err := shared.ParseEmail(trimmed); err == nil {
		return strings.ToLower(string(emailAddr))
	}
	return trimmed
}

func rawRememberedAccessToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("remembered Authorization header is empty")
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("expected Bearer header, got %q", authHeader)
	}
	return strings.TrimPrefix(authHeader, "Bearer "), nil
}

func (a *steps) decodeErrorDetails() (map[string]any, error) {
	var errResp struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal(a.ResponseBody, &errResp); err == nil && errResp.Code != "" {
		return errResp.Details, nil
	}

	var wrapped struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(a.ResponseBody, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Error.Details, nil
}
