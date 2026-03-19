package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type GetProfileInput struct {
	ID int64
}

type GetProfileOutput struct {
	ID        int64
	Name      string
	Avatar    string
	RoleCodes []string
}

type GetProfileUseCase struct {
	userRepo user.Repository
}

func NewGetProfileUseCase(
	userRepo user.Repository,
) *GetProfileUseCase {
	return &GetProfileUseCase{
		userRepo: userRepo,
	}
}

func (uc *GetProfileUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetProfileInput],
) (*GetProfileOutput, error) {

	userID := shared.UserID(input.Data.ID)
	userData, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	roles := make([]string, len(userData.RoleCodes))
	for i, role := range userData.RoleCodes {
		roles[i] = role.String()
	}

	return &GetProfileOutput{
		ID:        int64(userData.ID),
		Name:      userData.Name,
		Avatar:    userData.Avatar,
		RoleCodes: roles,
	}, nil
}
