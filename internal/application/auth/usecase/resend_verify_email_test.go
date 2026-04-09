package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

func resendVerifyEmailInput(email string) appShared.UseCaseInput[ResendVerifyEmailInput] {
	return appShared.UseCaseInput[ResendVerifyEmailInput]{Data: ResendVerifyEmailInput{Email: email}}
}

func newResendVerifyEmailUseCase(
	store *authVerificationStoreStub,
	accountRepo *authAccountRepoStub,
	emailService *authEmailServiceStub,
) *ResendVerifyEmailUseCase {
	return NewResendVerifyEmailUseCase(
		store,
		accountRepo,
		emailService,
		func(_, _, _ string) appEmail.EmailBuilder { return &authEmailBuilderStub{} },
	)
}

func TestResendVerifyEmailUseCase_SuccessHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	accountRepo := newAuthAccountRepoStub()
	accountRepo.accountsByEmail["applying@example.com"] = newExistingAuthAccount(101, shared.EmailAddress("applying@example.com"), "applying_account", account.Applying)
	emailService := &authEmailServiceStub{}
	uc := newResendVerifyEmailUseCase(store, accountRepo, emailService)
	input := resendVerifyEmailInput("applying@example.com")

	t.Log("Given: applying account exists for the requested email")
	t.Log("Input: email_present=true account_status=Applying")
	t.Log("Action: execute resend verify email success path")

	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Output: unexpected error=%v; Duration=%s", err, time.Since(start))
	}

	t.Logf("Output: out=%+v", out)
	t.Logf("Mutation: store_calls=%d delete_calls=%d email_send_calls=%d", store.storeCalls, store.deleteCalls, emailService.sendCalls)
	t.Logf("Duration: %s", time.Since(start))

	if store.storeCalls != 1 {
		t.Fatalf("expected store_calls=1, got %d", store.storeCalls)
	}
	if emailService.sendCalls != 1 {
		t.Fatalf("expected email_send_calls=1, got %d", emailService.sendCalls)
	}
	if store.deleteCalls != 0 {
		t.Fatalf("expected delete_calls=0 on success, got %d", store.deleteCalls)
	}
}

func TestResendVerifyEmailUseCase_AccountNotFoundHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	accountRepo := newAuthAccountRepoStub()
	emailService := &authEmailServiceStub{}
	uc := newResendVerifyEmailUseCase(store, accountRepo, emailService)
	input := resendVerifyEmailInput("unknown@example.com")

	t.Log("Given: no account registered for the requested email")
	t.Log("Input: email_present=true account_found=false")
	t.Log("Action: execute resend verify email account not found path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: store_calls=%d delete_calls=%d email_send_calls=%d", store.storeCalls, store.deleteCalls, emailService.sendCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if store.storeCalls != 0 || emailService.sendCalls != 0 {
		t.Fatalf("expected no store/email on account not found, got store=%d email=%d", store.storeCalls, emailService.sendCalls)
	}
}

func TestResendVerifyEmailUseCase_NonApplyingAccountHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	accountRepo := newAuthAccountRepoStub()
	accountRepo.accountsByEmail["active@example.com"] = newExistingAuthAccount(101, shared.EmailAddress("active@example.com"), "active_account", account.Active)
	emailService := &authEmailServiceStub{}
	uc := newResendVerifyEmailUseCase(store, accountRepo, emailService)
	input := resendVerifyEmailInput("active@example.com")

	t.Log("Given: account exists but status is not Applying")
	t.Log("Input: email_present=true account_status=Active")
	t.Log("Action: execute resend verify email non-applying path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: store_calls=%d delete_calls=%d email_send_calls=%d", store.storeCalls, store.deleteCalls, emailService.sendCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if store.storeCalls != 0 || emailService.sendCalls != 0 {
		t.Fatalf("expected no store/email for non-applying account, got store=%d email=%d", store.storeCalls, emailService.sendCalls)
	}
}

func TestResendVerifyEmailUseCase_EmailSendFailureCleansTokenHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAuthVerificationStoreStub()
	accountRepo := newAuthAccountRepoStub()
	accountRepo.accountsByEmail["applying@example.com"] = newExistingAuthAccount(101, shared.EmailAddress("applying@example.com"), "applying_account", account.Applying)
	emailService := &authEmailServiceStub{sendErr: errors.New("smtp unavailable")}
	uc := newResendVerifyEmailUseCase(store, accountRepo, emailService)
	input := resendVerifyEmailInput("applying@example.com")

	t.Log("Given: applying account exists but email send fails")
	t.Log("Input: email_present=true account_status=Applying send_err=true")
	t.Log("Action: execute resend verify email email send failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: store_calls=%d delete_calls=%d email_send_calls=%d", store.storeCalls, store.deleteCalls, emailService.sendCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if store.storeCalls != 1 {
		t.Fatalf("expected store_calls=1 before email failure, got %d", store.storeCalls)
	}
	if store.deleteCalls != 1 {
		t.Fatalf("expected delete_calls=1 to clean up token after email failure, got %d", store.deleteCalls)
	}
}
