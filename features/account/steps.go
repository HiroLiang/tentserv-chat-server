package account

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
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
	ctx.Step(`^I register an account with email "([^"]*)", account "([^"]*)", display name "([^"]*)", and password "([^"]*)"$`, s.iRegisterAnAccount)
	ctx.Step(`^I verify the registered email$`, s.iVerifyTheRegisteredEmail)
	ctx.Step(`^I verify the same email token again$`, s.iVerifyTheSameEmailTokenAgain)
	ctx.Step(`^I verify email with an empty token$`, s.iVerifyEmailWithAnEmptyToken)
	ctx.Step(`^I verify email with an invalid token$`, s.iVerifyEmailWithAnInvalidToken)
	ctx.Step(`^the verification token expires$`, s.theVerificationTokenExpires)
	ctx.Step(`^I resend the verification email to "([^"]*)"$`, s.iResendTheVerificationEmailTo)
	ctx.Step(`^the verify email response should be an HTML success page$`, s.theVerifyEmailResponseShouldBeAnHTMLSuccessPage)
	ctx.Step(`^the account registration mutation should include account, user, role, token, and email$`, s.theRegistrationMutationShouldBeComplete)
	ctx.Step(`^the account registration mutation should stop before account creation$`, s.theRegistrationMutationShouldStopBeforeAccountCreation)
	ctx.Step(`^the email verification mutation should activate the account and consume the token$`, s.theEmailVerificationMutationShouldActivateTheAccountAndConsumeTheToken)
	ctx.Step(`^the reused email verification token should remain consumed$`, s.theReusedEmailVerificationTokenShouldRemainConsumed)
	ctx.Step(`^the email verification mutation should not update an account$`, s.theEmailVerificationMutationShouldNotUpdateAnAccount)
	ctx.Step(`^the expired token verification should not activate the account$`, s.theExpiredTokenVerificationShouldNotActivateAccount)
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
	fmt.Printf("Mutation: accounts=%d users=%d role_assignments=%d tokens=%d emails=%d\n",
		len(a.deps.accountRepo.accountsByID), a.deps.userRepo.createCalls, a.deps.roleRepo.assignCalls, len(a.deps.store.tokens), a.deps.email.sendCalls)
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

func (a *steps) iVerifyTheRegisteredEmail() error {
	token := a.deps.store.lastStoredToken
	if token == "" {
		return fmt.Errorf("no verification token was stored")
	}
	return a.requestVerifyEmail("registered", token, true)
}

func (a *steps) iVerifyTheSameEmailTokenAgain() error {
	token := a.deps.store.lastStoredToken
	if token == "" {
		return fmt.Errorf("no verification token was stored")
	}
	return a.requestVerifyEmail("reused", token, true)
}

func (a *steps) iVerifyEmailWithAnEmptyToken() error {
	return a.requestVerifyEmail("empty", "", true)
}

func (a *steps) iVerifyEmailWithAnInvalidToken() error {
	return a.requestVerifyEmail("invalid", "invalid-token", true)
}

func (a *steps) theVerificationTokenExpires() error {
	a.start = time.Now()
	token := a.deps.store.lastStoredToken
	fmt.Println("Given: a valid token was issued but its TTL has elapsed")
	fmt.Printf("Input: token_present=%t\n", token != "")
	fmt.Println("Action: expire token in verification store (simulates Redis TTL)")
	a.deps.store.expireToken(token)
	fmt.Printf("Output: tokens_remaining=%d\n", len(a.deps.store.tokens))
	fmt.Printf("Mutation: token_expired=true delete_calls_unchanged=%d\n", a.deps.store.deleteCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iResendTheVerificationEmailTo(email string) error {
	a.start = time.Now()
	storeBefore := a.deps.store.storeCalls
	emailBefore := a.deps.email.sendCalls
	fmt.Println("Given: resend verification email HTTP endpoint is available")
	fmt.Printf("Input: email=%s store_calls_before=%d email_calls_before=%d\n", email, storeBefore, emailBefore)
	fmt.Println("Action: POST /api/auth/resend-verify-email")

	payload := map[string]string{"email": email}
	if err := a.DoJSONRequest(http.MethodPost, "/api/auth/resend-verify-email", payload); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: token_store_calls=%d email_send_calls=%d tokens_in_store=%d\n",
		a.deps.store.storeCalls, a.deps.email.sendCalls, len(a.deps.store.tokens))
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
	noActivation := len(a.deps.store.tokens) == 0

	status, accountFound := a.firstAccountStatus()
	notActive := !accountFound || status == domainaccount.Applying

	fmt.Printf("Output: no_verify_delete=%t no_activation=%t account_status=%s\n", noVerifyDelete, noActivation, status)
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d tokens_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.tokens))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !noVerifyDelete {
		return fmt.Errorf("expected no delete call from verify use case for expired token, got delete_calls=%d", a.deps.store.deleteCalls)
	}
	if !noActivation || !notActive {
		return fmt.Errorf("expected account to remain unactivated, got tokens_remaining=%d account_status=%s", len(a.deps.store.tokens), status)
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
	fmt.Printf("Mutation: token_store_calls=%d email_send_calls=%d tokens_in_store=%d\n",
		a.deps.store.storeCalls, a.deps.email.sendCalls, len(a.deps.store.tokens))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !tokenStored || !emailSent {
		return fmt.Errorf("expected resend to store a new token and send an email, got store_calls=%d email_calls=%d",
			a.deps.store.storeCalls, a.deps.email.sendCalls)
	}
	return nil
}

func (a *steps) requestVerifyEmail(caseName, token string, includeToken bool) error {
	a.start = time.Now()
	fmt.Println("Given: email verification HTTP endpoint is available")
	fmt.Printf("Input: case=%s token_present=%t token_consumed=%t\n", caseName, token != "", len(a.deps.store.tokens) == 0)
	fmt.Println("Action: GET /api/auth/verify-email")

	path := "/api/auth/verify-email"
	if includeToken {
		path += "?token=" + url.QueryEscape(token)
	}
	if err := a.DoRequest(http.MethodGet, path); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d content_type=%q body_len=%d\n",
		a.Response.StatusCode, a.Response.Header.Get("Content-Type"), len(a.ResponseBody))
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d tokens_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.tokens))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) theVerifyEmailResponseShouldBeAnHTMLSuccessPage() error {
	start := time.Now()
	fmt.Println("Given: verify email response should render a browser success page")
	fmt.Println("Input: expecting text/html and app deep link")
	fmt.Println("Action: inspect response headers and body")

	contentType := a.Response.Header.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html")
	hasDeepLink := bytes.Contains(a.ResponseBody, []byte("tentserv-chat://email-verified"))
	hasBrandCopy := bytes.Contains(a.ResponseBody, []byte("Tentserv Chat"))

	fmt.Printf("Output: is_html=%t has_deep_link=%t has_brand_copy=%t\n", isHTML, hasDeepLink, hasBrandCopy)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !isHTML {
		return fmt.Errorf("expected verify email response to be text/html, got %q", contentType)
	}
	if !hasDeepLink || !hasBrandCopy {
		return fmt.Errorf("expected verify email HTML success page, got body_len=%d", len(a.ResponseBody))
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
	tokenStored := a.deps.store.storeCalls == 1 && len(a.deps.store.tokens) == 1
	emailSent := a.deps.email.sendCalls == 1

	fmt.Printf("Output: account_created=%t user_created=%t role_assigned=%t token_stored=%t email_sent=%t\n",
		accountCreated, userCreated, roleAssigned, tokenStored, emailSent)
	fmt.Printf("Mutation: account_create_calls=%d user_create_calls=%d role_assign_calls=%d token_store_calls=%d email_send_calls=%d\n",
		a.deps.accountRepo.createCalls, a.deps.userRepo.createCalls, a.deps.roleRepo.assignCalls, a.deps.store.storeCalls, a.deps.email.sendCalls)
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
	tokenConsumed := len(a.deps.store.tokens) == 0 && a.deps.store.deleteCalls == 1
	mutationComplete := accountActive && tokenConsumed && a.deps.store.getCalls == 1 && a.deps.accountRepo.updateCalls == 2

	fmt.Printf("Output: account_found=%t account_active=%t token_consumed=%t mutation_complete=%t\n",
		accountFound, accountActive, tokenConsumed, mutationComplete)
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d tokens_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.tokens))
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
	stillConsumed := len(a.deps.store.tokens) == 0 && a.deps.store.getCalls == 2 && a.deps.store.deleteCalls == 1
	noSecondUpdate := a.deps.accountRepo.updateCalls == 2

	fmt.Printf("Output: account_active=%t token_still_consumed=%t no_second_update=%t\n",
		accountActive, stillConsumed, noSecondUpdate)
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d tokens_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.tokens))
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
	noMutation := noUpdate && noDelete && len(a.deps.store.tokens) == 0

	fmt.Printf("Output: no_account_update=%t no_token_delete=%t no_mutation=%t\n", noUpdate, noDelete, noMutation)
	fmt.Printf("Mutation: token_get_calls=%d token_delete_calls=%d account_update_calls=%d tokens_remaining=%d\n",
		a.deps.store.getCalls, a.deps.store.deleteCalls, a.deps.accountRepo.updateCalls, len(a.deps.store.tokens))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !noMutation {
		return fmt.Errorf("expected email verification rejection without account update")
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
