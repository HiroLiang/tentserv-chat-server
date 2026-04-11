package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

func verifyEmailInput(token string) appShared.UseCaseInput[VerifyEmailInput] {
	return appShared.UseCaseInput[VerifyEmailInput]{Data: VerifyEmailInput{Token: token, Code: "123456"}}
}

func seedVerificationSession(store *authVerificationStoreStub, token string, accountID int64) {
	store.sessions[token] = port.VerificationSession{
		AccountID:         accountID,
		Code:              "123456",
		ExpiresAtMS:       time.Now().Add(time.Minute).UnixMilli(),
		RemainingAttempts: verificationMaxAttempts,
	}
	store.accountTokens[accountID] = token
}

func seedExpiredVerificationSession(store *authVerificationStoreStub, token string, accountID int64) {
	store.sessions[token] = port.VerificationSession{
		AccountID:         accountID,
		Code:              "123456",
		ExpiresAtMS:       time.Now().Add(-time.Minute).UnixMilli(),
		RemainingAttempts: verificationMaxAttempts,
	}
	store.accountTokens[accountID] = token
}

func TestVerifyEmailUseCase_ActivatesApplyingAccountHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	seedVerificationSession(store, "valid-token", 101)
	accountRepo := newAuthAccountRepoStub()
	accountRepo.accountsByID[101] = newExistingAuthAccount(101, shared.EmailAddress("new@example.com"), "new_account", account.Applying)
	uc := NewVerifyEmailUseCase(store, accountRepo)
	input := verifyEmailInput("valid-token")

	t.Log("Given: verification token maps to an applying account")
	t.Log("Input: token_present=true")
	t.Log("Action: execute email verification activation path")

	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Output: unexpected error=%v; Duration=%s", err, time.Since(start))
	}

	t.Logf("Output: out=%+v account_status=%s", out, accountRepo.lastUpdated.Status)
	t.Logf("Mutation: get_calls=%d delete_calls=%d find_account_calls=%d update_calls=%d token_removed=%t",
		store.getCalls, store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls, len(store.sessions) == 0)
	t.Logf("Duration: %s", time.Since(start))

	if accountRepo.lastUpdated == nil || accountRepo.lastUpdated.Status != account.Active {
		t.Fatalf("expected account status active, got %+v", accountRepo.lastUpdated)
	}
	if store.deleteCalls != 1 || accountRepo.updateCalls != 1 {
		t.Fatalf("expected delete=1 update=1, got delete=%d update=%d", store.deleteCalls, accountRepo.updateCalls)
	}
}

func TestVerifyEmailUseCase_MissingTokenHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	accountRepo := newAuthAccountRepoStub()
	uc := NewVerifyEmailUseCase(store, accountRepo)
	input := verifyEmailInput("missing-token")

	t.Log("Given: verification token does not exist in store")
	t.Log("Input: token_present=true")
	t.Log("Action: execute email verification missing-token path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: get_calls=%d delete_calls=%d find_account_calls=%d update_calls=%d",
		store.getCalls, store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrTokenInvalid)
	if store.deleteCalls != 0 || accountRepo.findByIDCalls != 0 || accountRepo.updateCalls != 0 {
		t.Fatalf("expected no delete/find/update on missing token, got delete=%d find=%d update=%d", store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls)
	}
}

func TestVerifyEmailUseCase_StoreGetFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	store.getErr = errors.New("redis get failed")
	accountRepo := newAuthAccountRepoStub()
	uc := NewVerifyEmailUseCase(store, accountRepo)
	input := verifyEmailInput("valid-token")

	t.Log("Given: verification store returns an error on get")
	t.Log("Input: token_present=true")
	t.Log("Action: execute email verification store get failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: get_calls=%d delete_calls=%d find_account_calls=%d update_calls=%d",
		store.getCalls, store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrTokenInvalid)
}

func TestVerifyEmailUseCase_DeleteFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	seedVerificationSession(store, "valid-token", 101)
	store.deleteErr = errors.New("redis delete failed")
	accountRepo := newAuthAccountRepoStub()
	accountRepo.accountsByID[101] = newExistingAuthAccount(101, shared.EmailAddress("new@example.com"), "new_account", account.Applying)
	uc := NewVerifyEmailUseCase(store, accountRepo)
	input := verifyEmailInput("valid-token")

	t.Log("Given: verification token exists but delete fails before account activation")
	t.Log("Input: token_present=true")
	t.Log("Action: execute email verification delete failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: get_calls=%d delete_calls=%d find_account_calls=%d update_calls=%d",
		store.getCalls, store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if accountRepo.findByIDCalls != 0 || accountRepo.updateCalls != 0 {
		t.Fatalf("expected delete failure before account lookup/update, got find=%d update=%d", accountRepo.findByIDCalls, accountRepo.updateCalls)
	}
}

func TestVerifyEmailUseCase_AccountNotFoundHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	seedVerificationSession(store, "valid-token", 101)
	accountRepo := newAuthAccountRepoStub()
	uc := NewVerifyEmailUseCase(store, accountRepo)
	input := verifyEmailInput("valid-token")

	t.Log("Given: token exists but target account does not exist")
	t.Log("Input: token_present=true")
	t.Log("Action: execute email verification account not found path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: get_calls=%d delete_calls=%d find_account_calls=%d update_calls=%d token_removed=%t",
		store.getCalls, store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls, len(store.sessions) == 0)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrTokenInvalid)
	if store.deleteCalls != 1 || accountRepo.findByIDCalls != 1 || accountRepo.updateCalls != 0 {
		t.Fatalf("expected delete=1 find=1 update=0, got delete=%d find=%d update=%d", store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls)
	}
}

func TestVerifyEmailUseCase_NonApplyingAccountHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	seedVerificationSession(store, "valid-token", 101)
	accountRepo := newAuthAccountRepoStub()
	accountRepo.accountsByID[101] = newExistingAuthAccount(101, shared.EmailAddress("new@example.com"), "new_account", account.Active)
	uc := NewVerifyEmailUseCase(store, accountRepo)
	input := verifyEmailInput("valid-token")

	t.Log("Given: token maps to an account that is not applying")
	t.Log("Input: token_present=true")
	t.Log("Action: execute email verification invalid account status path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: get_calls=%d delete_calls=%d find_account_calls=%d update_calls=%d token_removed=%t",
		store.getCalls, store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls, len(store.sessions) == 0)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrTokenInvalid)
	if accountRepo.updateCalls != 0 {
		t.Fatalf("expected no update for non-applying account, got update=%d", accountRepo.updateCalls)
	}
}

func TestVerifyEmailUseCase_UpdateFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	seedVerificationSession(store, "valid-token", 101)
	accountRepo := newAuthAccountRepoStub()
	accountRepo.accountsByID[101] = newExistingAuthAccount(101, shared.EmailAddress("new@example.com"), "new_account", account.Applying)
	accountRepo.updateErr = errors.New("update failed")
	uc := NewVerifyEmailUseCase(store, accountRepo)
	input := verifyEmailInput("valid-token")

	t.Log("Given: token maps to applying account but account update fails")
	t.Log("Input: token_present=true")
	t.Log("Action: execute email verification update failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: get_calls=%d delete_calls=%d find_account_calls=%d update_calls=%d token_removed=%t",
		store.getCalls, store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls, len(store.sessions) == 0)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if accountRepo.updateCalls != 1 {
		t.Fatalf("expected update=1, got update=%d", accountRepo.updateCalls)
	}
}

func TestVerifyEmailUseCase_ExpiredTokenHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	seedExpiredVerificationSession(store, "expired-token", 101)
	accountRepo := newAuthAccountRepoStub()
	accountRepo.accountsByID[101] = newExistingAuthAccount(101, shared.EmailAddress("new@example.com"), "new_account", account.Applying)
	uc := NewVerifyEmailUseCase(store, accountRepo)
	input := verifyEmailInput("expired-token")

	t.Log("Given: verification token is still retained in store but its expires_at_ms is already in the past")
	t.Log("Input: token_present=true token_expired=true")
	t.Log("Action: execute email verification expired token path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: get_calls=%d delete_calls=%d find_account_calls=%d update_calls=%d",
		store.getCalls, store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrTokenInvalid)
	if store.deleteCalls != 1 || accountRepo.findByIDCalls != 0 || accountRepo.updateCalls != 0 {
		t.Fatalf("expected delete=1 find=0 update=0 for retained expired token, got delete=%d find=%d update=%d",
			store.deleteCalls, accountRepo.findByIDCalls, accountRepo.updateCalls)
	}
}

func TestVerifyEmailUseCase_ConcurrentTokenVerificationRaceHasStructuredLog(t *testing.T) {
	// This test verifies that concurrent calls to VerifyEmailUseCase produce no data races
	// (run with -race) and that the account ends up Active after any activation.
	//
	// Protection layers under concurrency:
	//   Layer 1 (Get timing): goroutines whose Get runs after a Delete sees ok=false → ErrTokenInvalid.
	//   Layer 2 (status check): goroutines that reach FindByID after an Update sees Active → ErrTokenInvalid.
	//
	// Because Get and Delete are separate operations, some goroutines may proceed past
	// both layers and attempt concurrent Updates. The account repo's mutex ensures
	// individual operations are safe; the final account state is always Active.
	start := time.Now()
	store := newAuthVerificationStoreStub()
	seedVerificationSession(store, "race-token", 101)
	accountRepo := newAuthAccountRepoStub()
	accountRepo.accountsByID[101] = newExistingAuthAccount(101, shared.EmailAddress("new@example.com"), "new_account", account.Applying)
	uc := NewVerifyEmailUseCase(store, accountRepo)

	t.Log("Given: one valid token maps to an applying account")
	t.Log("Input: token_present=true goroutines=10")
	t.Log("Action: execute email verification concurrently from 10 goroutines using the same token")

	const n = 10
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, err := uc.Execute(context.Background(), verifyEmailInput("race-token"))
			errs[i] = err
		}()
	}
	wg.Wait()

	// All goroutines finished — no concurrent access below.
	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}

	var lastStatus account.Status
	if accountRepo.lastUpdated != nil {
		lastStatus = accountRepo.lastUpdated.Status
	}

	t.Logf("Output: successes=%d update_calls=%d account_status=%s", successes, accountRepo.updateCalls, lastStatus)
	t.Logf("Mutation: get_calls=%d delete_calls=%d update_calls=%d", store.getCalls, store.deleteCalls, accountRepo.updateCalls)
	t.Logf("Duration: %s", time.Since(start))

	if successes < 1 {
		t.Fatalf("expected at least 1 successful activation, got 0")
	}
	if lastStatus != account.Active {
		t.Fatalf("expected account status Active after concurrent activations, got %s", lastStatus)
	}
	if store.deleteCalls < 1 {
		t.Fatalf("expected at least 1 delete call, got %d", store.deleteCalls)
	}
}
