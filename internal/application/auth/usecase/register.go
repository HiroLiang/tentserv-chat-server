package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/application/shared/email"
	"github.com/HiroLiang/goat-server/internal/application/shared/security"
	"github.com/HiroLiang/goat-server/internal/domain/account"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/domain/transaction"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	infraBuilder "github.com/HiroLiang/goat-server/internal/infrastructure/email/builder"
	"github.com/gofrs/uuid"
)

type RegisterInput struct {
	Name     string
	Account  string
	Email    string
	Password string
}

type RegisterOutput struct {
	ID int64
}

type RegisterUseCase struct {
	uow          transaction.UnitOfWork
	hasher       security.Hasher
	accountRepo  account.Repository
	userRepo     user.Repository
	emailService email.EmailService
}

func NewRegisterUseCase(
	uow transaction.UnitOfWork,
	hasher security.Hasher,
	accountRepo account.Repository,
	userRepo user.Repository,
) *RegisterUseCase {
	return &RegisterUseCase{
		uow:         uow,
		hasher:      hasher,
		accountRepo: accountRepo,
		userRepo:    userRepo,
	}
}

func (uc *RegisterUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[RegisterInput],
) (RegisterOutput, error) {
	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Create Account
	emailAddr, err := shared.ParseEmail(input.Data.Email)
	if err != nil {
		return RegisterOutput{}, ErrInvalidEmail
	}

	// Hash password
	hash, err := uc.hasher.Hash(input.Data.Password)
	if err != nil {
		return RegisterOutput{}, ErrInvalidPassword
	}

	// Generate public ID
	publicID, err := uuid.NewV4()
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	// Create Account
	newAccount := account.NewAccount(
		publicID,
		emailAddr,
		input.Data.Account,
		hash,
		1,
	)
	accountId, err := uc.accountRepo.Create(ctx, newAccount)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrAccountExist):
			return RegisterOutput{}, ErrAccountExist
		case errors.Is(err, account.ErrEmailExist):
			return RegisterOutput{}, ErrEmailExist
		default:
			return RegisterOutput{}, ErrRegisterFailed
		}
	}

	// Send verification email
	builder := infraBuilder.NewRegisterMailBuilder()
	err = uc.emailService.Send(ctx, builder)
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	return RegisterOutput{int64(accountId)}, tx.Commit()
}
