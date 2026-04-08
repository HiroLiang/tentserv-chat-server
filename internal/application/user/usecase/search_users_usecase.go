package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

const defaultSearchLimit = 50

var ErrInvalidSearchInput = errors.New("exactly one search parameter is required")

type SearchUsersInput struct {
	Name     string
	Account  string
	PublicID string
	Limit    int
	Offset   int
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

	limit := d.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	offset := d.Offset
	if offset < 0 {
		offset = 0
	}

	var (
		results []*user.UserSearchResult
		err     error
	)
	switch {
	case d.Name != "":
		results, err = uc.userRepo.SearchByName(ctx, d.Name, limit, offset)
	case d.Account != "":
		results, err = uc.userRepo.FindByAccountName(ctx, d.Account, limit, offset)
	default:
		results, err = uc.userRepo.FindByPublicID(ctx, d.PublicID, limit, offset)
	}
	if err != nil {
		return nil, err
	}

	var currentUserID = input.Base.Auth.UserID

	// Fetch all friendships for current user in one query to avoid N+1
	friendshipMap := uc.buildFriendshipMap(ctx, currentUserID)

	out := make([]*UserSearchResultWithStatus, 0, len(results))
	for _, r := range results {
		if r.ID == currentUserID {
			continue
		}
		item := &UserSearchResultWithStatus{UserSearchResult: r}
		if status, ok := friendshipMap[r.ID]; ok {
			s := string(status)
			item.FriendshipStatus = &s
		}
		out = append(out, item)
	}

	return &SearchUsersOutput{Users: out}, nil
}

// buildFriendshipMap returns a map of otherUserID → friendship.Status for all friendships
// involving the current user. Errors are silently ignored (friendship status is best-effort).
func (uc *SearchUsersUseCase) buildFriendshipMap(ctx context.Context, userID shared.UserID) map[shared.UserID]friendship.Status {
	result := make(map[shared.UserID]friendship.Status)
	friendships, err := uc.friendshipRepo.FindAllByUserID(ctx, userID)
	if err != nil {
		return result
	}
	for _, f := range friendships {
		other := f.FriendID
		if f.UserID != userID {
			other = f.UserID
		}
		result[other] = f.Status
	}
	return result
}
