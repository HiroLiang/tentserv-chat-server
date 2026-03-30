package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
)

type UpdateProfileInput struct {
	Name      string
	RoleCodes []string
}

type UpdateProfileOutput struct {
	ID        int64
	Name      string
	Avatar    string
	RoleCodes []string
}

type UpdateProfileUseCase struct {
	userRepo     user.Repository
	userRoleRepo userrole.Repository
}

func NewUpdateProfileUseCase(userRepo user.Repository, userRoleRepo userrole.Repository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{
		userRepo:     userRepo,
		userRoleRepo: userRoleRepo,
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

	nameChanged := false
	if userData.Name != input.Data.Name {
		userData.Name = input.Data.Name
		nameChanged = true
	}

	rolesChanged := false
	var newCodes []role.Code
	if len(input.Data.RoleCodes) > 0 {
		newCodes = make([]role.Code, len(input.Data.RoleCodes))
		for i, code := range input.Data.RoleCodes {
			newCodes[i], err = role.CodeFrom(code)
			if err != nil {
				return nil, ErrInvalidRoleCode
			}
		}
		rolesChanged = true
	}

	if nameChanged {
		if err = u.userRepo.Update(ctx, userData); err != nil {
			return nil, ErrUpdateProfile
		}
	}

	if rolesChanged {
		// Revoke all existing roles then assign the new set via the join table
		existing, err := u.userRoleRepo.FindRolesByUser(ctx, userData.ID)
		if err != nil {
			return nil, ErrUpdateProfile
		}
		for _, r := range existing {
			if err := u.userRoleRepo.Revoke(ctx, userData.ID, r.Code); err != nil {
				return nil, ErrUpdateProfile
			}
		}
		for _, code := range newCodes {
			if err := u.userRoleRepo.Assign(ctx, userData.ID, code); err != nil {
				return nil, ErrUpdateProfile
			}
		}
		userData.RoleCodes = newCodes
	}

	roleCodes := make([]string, len(userData.RoleCodes))
	for i, c := range userData.RoleCodes {
		roleCodes[i] = string(c)
	}

	return &UpdateProfileOutput{
		ID:        int64(userData.ID),
		Name:      userData.Name,
		Avatar:    userData.Avatar,
		RoleCodes: roleCodes,
	}, nil
}
