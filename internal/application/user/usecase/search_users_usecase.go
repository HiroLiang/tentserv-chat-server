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
	BlockedBy        *string
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

	// Fetch all friendships for current user in one query to avoid N+1.
	friendshipMap := uc.buildFriendshipMap(ctx, currentUserID)

	out := make([]*UserSearchResultWithStatus, 0, len(results))
	for _, r := range results {
		if r.ID == currentUserID {
			continue
		}
		item := &UserSearchResultWithStatus{UserSearchResult: r}
		if view, ok := friendshipMap[r.ID]; ok {
			if view.OtherToCurrent != nil && *view.OtherToCurrent == friendship.StatusBlocked {
				continue
			}
			status := view.CurrentToOther
			if status == nil {
				status = view.OtherToCurrent
			}
			if status == nil {
				out = append(out, item)
				continue
			}
			s := string(*status)
			item.FriendshipStatus = &s
			if *status == friendship.StatusBlocked && view.CurrentToOther != nil && *view.CurrentToOther == friendship.StatusBlocked {
				blockedBy := "me"
				item.BlockedBy = &blockedBy
			}
		}
		out = append(out, item)
	}

	return &SearchUsersOutput{Users: out}, nil
}

type searchFriendshipView struct {
	CurrentToOther *friendship.Status
	OtherToCurrent *friendship.Status
}

// buildFriendshipMap returns a map of otherUserID → directed statuses for all
// friendships involving the current user. Errors are silently ignored because
// friendship status is best-effort for search.
func (uc *SearchUsersUseCase) buildFriendshipMap(ctx context.Context, userID shared.UserID) map[shared.UserID]searchFriendshipView {
	result := make(map[shared.UserID]searchFriendshipView)
	friendships, err := uc.friendshipRepo.FindAllByUserID(ctx, userID)
	if err != nil {
		return result
	}
	for _, f := range friendships {
		status := f.Status
		if f.UserID == userID {
			view := result[f.FriendID]
			view.CurrentToOther = &status
			result[f.FriendID] = view
			continue
		}
		if f.FriendID == userID {
			view := result[f.UserID]
			view.OtherToCurrent = &status
			result[f.UserID] = view
		}
	}
	return result
}
