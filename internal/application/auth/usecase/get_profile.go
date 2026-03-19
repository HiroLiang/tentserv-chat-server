package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type GetProfileInput struct{}

type GetProfileOutput struct {
	PublicID    string
	Email       string
	AccountName string
	Status      string
	UserISs     []int64
	CurrentUser UserProfile
}

type UserProfile struct {
	ID        int64
	Name      string
	Avatar    string
	RoleCodes []string
}

type GetProfileUseCase struct {
	accountRepo account.Repository
	userRepo    user.Repository
}

func NewGetProfileUseCase(
	accountRepo account.Repository,
	userRepo user.Repository,
) *GetProfileUseCase {
	return &GetProfileUseCase{
		accountRepo: accountRepo,
		userRepo:    userRepo,
	}
}

func (uc *GetProfileUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetProfileInput],
) (*GetProfileOutput, error) {
	accountID := input.Base.Auth.AccountID

	accountData, err := uc.accountRepo.FindByID(ctx, accountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	currentUser, err := uc.userRepo.FindByID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	userIDs := make([]int64, len(accountData.UserIDs))
	for i, id := range accountData.UserIDs {
		userIDs[i] = int64(id)
	}

	roles := make([]string, len(currentUser.RoleCodes))
	for i, role := range currentUser.RoleCodes {
		roles[i] = role.String()
	}

	return &GetProfileOutput{
		PublicID:    accountData.PublicID.String(),
		Email:       string(accountData.Email),
		AccountName: accountData.AccountName,
		Status:      string(accountData.Status),
		UserISs:     userIDs,
		CurrentUser: UserProfile{
			ID:        int64(currentUser.ID),
			Name:      currentUser.Name,
			Avatar:    currentUser.Avatar,
			RoleCodes: roles,
		},
	}, nil
}
