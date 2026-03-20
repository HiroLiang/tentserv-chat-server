package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

var ErrInvalidSearchInput = errors.New("exactly one search parameter is required")

type SearchUsersInput struct {
	Name     string
	Account  string
	PublicID string
}

type UserSearchResultWithStatus struct {
	*user.UserSearchResult
	FriendshipStatus *string
}

type SearchUsersOutput struct {
	Users []*UserSearchResultWithStatus
}

type SearchUsersUseCase struct {
	userRepo       user.Repository
	friendshipRepo friendship.Repository
}

func NewSearchUsersUseCase(userRepo user.Repository, friendshipRepo friendship.Repository) *SearchUsersUseCase {
	return &SearchUsersUseCase{userRepo: userRepo, friendshipRepo: friendshipRepo}
}

func (uc *SearchUsersUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[SearchUsersInput],
) (*SearchUsersOutput, error) {
	d := input.Data

	count := 0
	if d.Name != "" {
		count++
	}
	if d.Account != "" {
		count++
	}
	if d.PublicID != "" {
		count++
	}
	if count != 1 {
		return nil, ErrInvalidSearchInput
	}

	var (
		results []*user.UserSearchResult
		err     error
	)
	switch {
	case d.Name != "":
		results, err = uc.userRepo.SearchByName(ctx, d.Name)
	case d.Account != "":
		results, err = uc.userRepo.FindByAccountName(ctx, d.Account)
	default:
		results, err = uc.userRepo.FindByPublicID(ctx, d.PublicID)
	}
	if err != nil {
		return nil, err
	}

	var currentUserID = input.Base.Auth.UserID

	out := make([]*UserSearchResultWithStatus, 0, len(results))
	for _, r := range results {
		item := &UserSearchResultWithStatus{UserSearchResult: r}
		f, ferr := uc.friendshipRepo.FindBetweenUsers(ctx, currentUserID, r.ID)
		if ferr == nil {
			s := string(f.Status)
			item.FriendshipStatus = &s
		}
		out = append(out, item)
	}

	return &SearchUsersOutput{Users: out}, nil
}
