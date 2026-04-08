package account

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/cucumber/godog"
)

type steps struct {
	*bddsupport.APITestContext
	deps  *Deps
	start time.Time
}

func RegisterSteps(ctx *godog.ScenarioContext, apiCtx *bddsupport.APITestContext, deps *Deps) {
	s := &steps{APITestContext: apiCtx, deps: deps}

	ctx.Step(`^account registration state is clean$`, s.accountRegistrationStateIsClean)
	ctx.Step(`^an account exists with email "([^"]*)" and account "([^"]*)"$`, s.anAccountExists)
	ctx.Step(`^I register an account with email "([^"]*)", account "([^"]*)", display name "([^"]*)", and password "([^"]*)"$`, s.iRegisterAnAccount)
	ctx.Step(`^I verify the registered email$`, s.iVerifyTheRegisteredEmail)
	ctx.Step(`^I verify the same email token again$`, s.iVerifyTheSameEmailTokenAgain)
	ctx.Step(`^I verify email with an empty token$`, s.iVerifyEmailWithAnEmptyToken)
	ctx.Step(`^I verify email with an invalid token$`, s.iVerifyEmailWithAnInvalidToken)
	ctx.Step(`^the verify email response should be an HTML success page$`, s.theVerifyEmailResponseShouldBeAnHTMLSuccessPage)
	ctx.Step(`^the account registration mutation should include account, user, role, token, and email$`, s.theRegistrationMutationShouldBeComplete)
	ctx.Step(`^the account registration mutation should stop before account creation$`, s.theRegistrationMutationShouldStopBeforeAccountCreation)
	ctx.Step(`^the email verification mutation should activate the account and consume the token$`, s.theEmailVerificationMutationShouldActivateTheAccountAndConsumeTheToken)
	ctx.Step(`^the reused email verification token should remain consumed$`, s.theReusedEmailVerificationTokenShouldRemainConsumed)
	ctx.Step(`^the email verification mutation should not update an account$`, s.theEmailVerificationMutationShouldNotUpdateAnAccount)
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

func (a *steps) firstAccountStatus() (domainaccount.Status, bool) {
	a.deps.accountRepo.mu.Lock()
	defer a.deps.accountRepo.mu.Unlock()

	for _, acc := range a.deps.accountRepo.accountsByID {
		return acc.Status, true
	}
	return "", false
}
