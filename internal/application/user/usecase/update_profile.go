package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type UpdateProfileInput struct {
	Name      string
	RoleCodes []string
}

type UpdateProfileOutput struct{}

type UpdateProfileUseCase struct {
	userRepo user.Repository
}

func NewUpdateProfileUseCase(userRepo user.Repository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{
		userRepo: userRepo,
	}
}

func (u *UpdateProfileUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[UpdateProfileInput]) (*UpdateProfileOutput, error,
) {
	userData, err := u.userRepo.FindByID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	isChanged := false

	if userData.Name != input.Data.Name {
		userData.Name = input.Data.Name
		isChanged = true
	}

	if input.Data.RoleCodes != nil && len(input.Data.RoleCodes) > 0 {
		codes := make([]role.Code, len(input.Data.RoleCodes))
		for i, code := range input.Data.RoleCodes {
			codes[i], err = role.CodeFrom(code)
			if err != nil {
				return nil, ErrInvalidRoleCode
			}
		}

		userData.RoleCodes = codes
		isChanged = true
	}

	if isChanged {
		err = u.userRepo.Update(ctx, userData)
		if err != nil {
			return nil, ErrUpdateProfile
		}
	}

	return &UpdateProfileOutput{}, nil
}
