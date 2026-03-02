package user

import (
	"context"
	"errors"
	"strconv"

	"github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/application/shared/auth"
	"github.com/HiroLiang/goat-server/internal/application/shared/security"
	"github.com/HiroLiang/goat-server/internal/application/shared/storage"
	session "github.com/HiroLiang/goat-server/internal/domain/auth"
	"github.com/HiroLiang/goat-server/internal/domain/device"
	"github.com/HiroLiang/goat-server/internal/domain/role"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/domain/userrole"
	"github.com/HiroLiang/goat-server/internal/shared/timeutil"
)

type UseCase struct {
	storageURL   string
	userRepo     user.Repository
	userRoleRepo userrole.Repository
	hasher       security.Hasher
	tokenService auth.TokenService
	fileStorage  storage.FileStorage
	deviceRepo   device.Repository
}

func NewUseCase(
	storageURL string,
	repo user.Repository,
	userRoleRepo userrole.Repository,
	hasher security.Hasher,
	tokenService auth.TokenService,
	fileStorage storage.FileStorage,
	deviceRepo device.Repository) *UseCase {
	return &UseCase{
		storageURL:   storageURL,
		userRepo:     repo,
		userRoleRepo: userRoleRepo,
		hasher:       hasher,
		tokenService: tokenService,
		fileStorage:  fileStorage,
		deviceRepo:   deviceRepo,
	}
}

// Register User register
func (u *UseCase) Register(ctx context.Context, input shared.UseCaseInput[RegisterInput]) error {
	hash, err := u.hasher.Hash(input.Data.Password)
	if err != nil {
		return user.ErrInvalidPassword
	}

	email, err := user.ParseEmail(input.Data.Email)
	if err != nil {
		return user.ErrInvalidEmail
	}

	newUser := user.NewUser(
		input.Data.Name,
		email,
		hash,
		input.Base.Request.IP,
	)

	if err := u.userRepo.Create(ctx, newUser); err != nil {
		return user.ErrUserAlreadyExists
	}

	return nil
}

// Login User login
func (u *UseCase) Login(ctx context.Context, input shared.UseCaseInput[LoginInput]) (LoginOutput, error) {

	// Check email and build vo
	email, err := user.ParseEmail(input.Data.Email)
	if err != nil {
		return LoginOutput{}, user.ErrInvalidEmail
	}

	// Check is user exists
	currentUser, err := u.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return LoginOutput{}, user.ErrUserNotFound
	}

	switch currentUser.Status {
	case user.Active:
		break
	case user.Applying:
		return LoginOutput{}, user.ErrUserApplying
	case user.Banned:
		return LoginOutput{}, user.ErrUserBanned
	default:
		return LoginOutput{}, user.ErrInvalidUser
	}

	// Check password
	if !u.hasher.Verify(input.Data.Password, currentUser.Password) {
		return LoginOutput{}, user.ErrInvalidPassword
	}

	// Update last login ip
	if currentUser.LastIP != input.Base.Request.IP {
		currentUser.LastIP = input.Base.Request.IP
		err = u.userRepo.Update(ctx, currentUser)
		if err != nil {
			return LoginOutput{}, err
		}
	}

	// Generate auth token and store in redis
	authToken, err := u.tokenService.Generate(ctx, session.CreateSessionParams{
		UserID:    strconv.FormatInt(int64(currentUser.ID), 10),
		IP:        input.Base.Request.IP,
		UserAgent: "",
	})
	if err != nil {
		return LoginOutput{}, user.ErrGenerateToken
	}

	// Bind the device to the user if device_id is provided
	if input.Data.DeviceID != "" {
		deviceID := device.ID(input.Data.DeviceID)
		if _, err := u.deviceRepo.FindByID(ctx, deviceID); err == nil {
			// Device exists — bind it; ignore bind errors silently
			_ = u.deviceRepo.BindUser(ctx, deviceID, currentUser.ID)
		}
	}

	return LoginOutput{Token: authToken}, nil
}

// Logout User logout
func (u *UseCase) Logout(ctx context.Context, input shared.UseCaseInput[struct{}]) error {
	return u.tokenService.Revoke(ctx, input.Base.Auth.Token)
}

// CurrentUserInfo Get current user info
func (u *UseCase) CurrentUserInfo(
	ctx context.Context,
	input shared.UseCaseInput[struct{}]) (CurrentUserOutput, error) {
	id, err := user.ParseID(input.Base.Auth.UserID)
	if err != nil {
		return CurrentUserOutput{}, user.ErrInvalidUser
	}

	domainUser, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return CurrentUserOutput{}, user.ErrUserNotFound
	}

	return CurrentUserOutput{
		ID:        int(domainUser.ID),
		Name:      domainUser.Name,
		Email:     string(domainUser.Email),
		AvatarURL: u.buildAvatarURL(domainUser.AvatarName),
		CreateAt:  timeutil.Format(domainUser.CreatedAt, "2006/01/02 15:04:05"),
	}, nil
}

// UpdateProfile updates the current user's display name.
func (u *UseCase) UpdateProfile(
	ctx context.Context,
	input shared.UseCaseInput[UpdateProfileInput]) error {
	id, err := user.ParseID(input.Base.Auth.UserID)
	if err != nil {
		return user.ErrInvalidUser
	}

	domainUser, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return user.ErrUserNotFound
	}

	domainUser.Name = input.Data.Name
	return u.userRepo.Update(ctx, domainUser)
}

// UploadAvatar processes and saves an avatar image for the current user.
// The caller is responsible for validating the image MIME type and size before calling this.
func (u *UseCase) UploadAvatar(
	ctx context.Context,
	input shared.UseCaseInput[UploadAvatarInput]) (UploadAvatarOutput, error) {
	id, err := user.ParseID(input.Base.Auth.UserID)
	if err != nil {
		return UploadAvatarOutput{}, user.ErrInvalidUser
	}

	domainUser, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return UploadAvatarOutput{}, user.ErrUserNotFound
	}

	fileName, err := u.fileStorage.SaveAvatar(ctx, int64(id), input.Data.Image, domainUser.AvatarName)
	if err != nil {
		return UploadAvatarOutput{}, err
	}

	domainUser.AvatarName = fileName
	if err := u.userRepo.Update(ctx, domainUser); err != nil {
		return UploadAvatarOutput{}, err
	}

	return UploadAvatarOutput{AvatarURL: u.buildAvatarURL(fileName)}, nil
}

// FindRolesByUser Find all roles of user
func (u *UseCase) FindRolesByUser(
	ctx context.Context,
	input shared.UseCaseInput[FindUserRolesInput]) (FindUserRolesOutput, error) {
	roles, err := u.userRoleRepo.FindRolesByUser(ctx, input.Data.UserID)
	if err != nil {
		return FindUserRolesOutput{}, err
	}

	types := make([]role.Type, len(roles))
	for _, r := range roles {
		types = append(types, r.Type)
	}

	return FindUserRolesOutput{Roles: types}, nil

}

// AssignRoleToUser Assign a role to the user
func (u *UseCase) AssignRoleToUser(
	ctx context.Context,
	input shared.UseCaseInput[AssignRoleInput]) error {
	if err := u.userRoleRepo.Assign(ctx, input.Data.UserID, input.Data.Role); err != nil {
		if errors.Is(err, userrole.ErrUserRoleAlreadyAssigned) {
			return err
		}

		return userrole.ErrAssignFailed
	}
	return nil
}

// RevokeRoleFromUser Revoke a role from the user
func (u *UseCase) RevokeRoleFromUser(ctx context.Context, input shared.UseCaseInput[RevokeRoleInput]) error {
	if err := u.userRoleRepo.Revoke(ctx, input.Data.UserID, input.Data.Role); err != nil {
		return err
	}
	return nil
}

func (u *UseCase) buildAvatarURL(fileName string) string {
	if fileName == "" {
		return ""
	}
	return u.storageURL + "/avatars/" + fileName
}
