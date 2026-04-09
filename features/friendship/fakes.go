package friendship

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	chatUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/usecase"
	friendshipUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/friendship/usecase"
	userUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/user/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	domainfriendship "github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type Deps struct {
	friendshipRepo  *bddFriendshipRepo
	userRepo        *bddUserRepo
	participantRepo *bddParticipantRepo
	chatRoomRepo    *bddChatRoomRepo
	chatMemberRepo  *bddChatMemberRepo
	invitationRepo  *bddChatInvitationRepo
}

type UseCases struct {
	SearchUsers       *userUseCase.SearchUsersUseCase
	GetFriends        *friendshipUseCase.GetFriendsUseCase
	GetBlockedUsers   *friendshipUseCase.GetBlockedUsersUseCase
	ApplyFriendship   *friendshipUseCase.ApplyFriendshipUseCase
	AcceptFriendship  *friendshipUseCase.AcceptFriendshipUseCase
	GetFriendRequests *friendshipUseCase.GetFriendRequestsUseCase
	RemoveFriendship  *friendshipUseCase.RemoveFriendshipUseCase
	GetSentRequests   *friendshipUseCase.GetSentRequestsUseCase
	CancelSentRequest *friendshipUseCase.CancelSentRequestUseCase
	BlockUser         *friendshipUseCase.BlockUserUseCase
	UnblockUser       *friendshipUseCase.UnblockUserUseCase
}

func NewDeps() *Deps {
	chatMemberRepo := newBDDChatMemberRepo()
	deps := &Deps{
		friendshipRepo:  newBDDFriendshipRepo(),
		userRepo:        newBDDUserRepo(),
		participantRepo: newBDDParticipantRepo(),
		chatRoomRepo:    newBDDChatRoomRepo(chatMemberRepo),
		chatMemberRepo:  chatMemberRepo,
		invitationRepo:  newBDDChatInvitationRepo(),
	}
	deps.Reset()
	return deps
}

func (d *Deps) Reset() {
	d.friendshipRepo.reset()
	d.userRepo.reset()
	d.participantRepo.reset()
	d.chatRoomRepo.reset()
	d.chatMemberRepo.reset()
	d.invitationRepo.reset()
}

func (d *Deps) RegisterUseCases(uow transaction.UnitOfWork) UseCases {
	return UseCases{
		SearchUsers:       userUseCase.NewSearchUsersUseCase(d.userRepo, d.friendshipRepo),
		GetFriends:        friendshipUseCase.NewGetFriendsUseCase(d.friendshipRepo, d.userRepo),
		GetBlockedUsers:   friendshipUseCase.NewGetBlockedUsersUseCase(d.friendshipRepo, d.userRepo),
		ApplyFriendship:   friendshipUseCase.NewApplyFriendshipUseCase(d.friendshipRepo),
		AcceptFriendship:  friendshipUseCase.NewAcceptFriendshipUseCase(uow, d.friendshipRepo, d.participantRepo, d.chatRoomRepo, d.chatMemberRepo),
		GetFriendRequests: friendshipUseCase.NewGetFriendRequestsUseCase(d.friendshipRepo, d.userRepo),
		RemoveFriendship:  friendshipUseCase.NewRemoveFriendshipUseCase(d.friendshipRepo),
		GetSentRequests:   friendshipUseCase.NewGetSentRequestsUseCase(d.friendshipRepo, d.userRepo),
		CancelSentRequest: friendshipUseCase.NewCancelSentRequestUseCase(d.friendshipRepo),
		BlockUser:         friendshipUseCase.NewBlockUserUseCase(d.friendshipRepo),
		UnblockUser:       friendshipUseCase.NewUnblockUserUseCase(d.friendshipRepo),
	}
}

func (d *Deps) RegisterChatUseCase(uow transaction.UnitOfWork) *chatUseCase.CreateChatRoomUseCase {
	return chatUseCase.NewCreateChatRoomUseCase(
		uow,
		d.chatRoomRepo,
		d.chatMemberRepo,
		d.participantRepo,
		d.friendshipRepo,
		d.invitationRepo,
	)
}

func (d *Deps) SeedUser(userID shared.UserID, name, avatar, accountName, publicID string) {
	d.userRepo.seed(userID, name, avatar, accountName, publicID)
	// Auto-seed a participant so accept-friendship can create the direct room.
	d.participantRepo.seedForUser(userID)
}

func (d *Deps) SeedFriendship(userID, friendID shared.UserID, status domainfriendship.Status) *domainfriendship.Friendship {
	return d.friendshipRepo.seed(userID, friendID, status)
}

func (d *Deps) FindFriendship(userID, friendID shared.UserID) (*domainfriendship.Friendship, bool) {
	return d.friendshipRepo.findPair(userID, friendID)
}

func (d *Deps) FriendshipCounts() (pending, accepted, blocked int) {
	return d.friendshipRepo.statusCounts()
}

// FindDirectRoomBetweenUsers returns the direct room (if any) created between two user IDs.
func (d *Deps) FindDirectRoomBetweenUsers(userID1, userID2 shared.UserID) (*chatroom.ChatRoom, bool) {
	p1 := d.participantRepo.findByUserID(userID1)
	p2 := d.participantRepo.findByUserID(userID2)
	if p1 == nil || p2 == nil {
		return nil, false
	}
	return d.chatRoomRepo.findDirect(p1.ID, p2.ID)
}

func (d *Deps) DirectRoomCountBetweenUsers(userID1, userID2 shared.UserID) int {
	p1 := d.participantRepo.findByUserID(userID1)
	p2 := d.participantRepo.findByUserID(userID2)
	if p1 == nil || p2 == nil {
		return 0
	}
	return d.chatRoomRepo.countDirectRooms(p1.ID, p2.ID)
}

// FindMembersByRoom returns all members of a room.
func (d *Deps) FindMembersByRoom(roomID chatroom.ID) []*chatmember.ChatMember {
	return d.chatMemberRepo.findByRoom(roomID)
}

// ─── User repo ───────────────────────────────────────────────────────────────

type bddUserRecord struct {
	user        *domainuser.User
	accountName string
	publicID    string
}

type bddUserRepo struct {
	mu        sync.Mutex
	usersByID map[shared.UserID]bddUserRecord
}

func newBDDUserRepo() *bddUserRepo {
	return &bddUserRepo{}
}

func (r *bddUserRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usersByID = map[shared.UserID]bddUserRecord{}
}

func (r *bddUserRepo) seed(userID shared.UserID, name, avatar, accountName, publicID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usersByID[userID] = bddUserRecord{
		user: &domainuser.User{
			ID:        userID,
			AccountID: shared.AccountID(userID),
			Name:      name,
			Avatar:    avatar,
			RoleCodes: []role.Code{role.User},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		accountName: accountName,
		publicID:    publicID,
	}
}

func (r *bddUserRepo) Create(_ context.Context, u *domainuser.User) (shared.UserID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u.ID == 0 {
		u.ID = shared.UserID(len(r.usersByID) + 1)
	}
	r.usersByID[u.ID] = bddUserRecord{user: cloneBDDUser(u)}
	return u.ID, nil
}

func (r *bddUserRepo) FindByID(_ context.Context, id shared.UserID) (*domainuser.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.usersByID[id]
	if !ok {
		return nil, domainuser.ErrUserNotFound
	}
	return cloneBDDUser(rec.user), nil
}

func (r *bddUserRepo) FindByAccountID(context.Context, shared.AccountID) (*[]domainuser.User, error) {
	return nil, nil
}

func (r *bddUserRepo) Update(context.Context, *domainuser.User) error {
	return nil
}

func (r *bddUserRepo) SearchByName(_ context.Context, keyword string, limit, offset int) ([]*domainuser.UserSearchResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	needle := strings.ToLower(keyword)
	items := make([]bddUserRecord, 0, len(r.usersByID))
	for _, rec := range r.usersByID {
		if strings.Contains(strings.ToLower(rec.user.Name), needle) {
			items = append(items, rec)
		}
	}
	return paginateUserRecords(items, limit, offset), nil
}

func (r *bddUserRepo) FindByAccountName(_ context.Context, accountName string, limit, offset int) ([]*domainuser.UserSearchResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]bddUserRecord, 0, 1)
	for _, rec := range r.usersByID {
		if rec.accountName == accountName {
			items = append(items, rec)
		}
	}
	return paginateUserRecords(items, limit, offset), nil
}

func (r *bddUserRepo) FindByPublicID(_ context.Context, publicID string, limit, offset int) ([]*domainuser.UserSearchResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]bddUserRecord, 0, 1)
	for _, rec := range r.usersByID {
		if rec.publicID == publicID {
			items = append(items, rec)
		}
	}
	return paginateUserRecords(items, limit, offset), nil
}

func paginateUserRecords(items []bddUserRecord, limit, offset int) []*domainuser.UserSearchResult {
	sort.Slice(items, func(i, j int) bool {
		return items[i].user.ID < items[j].user.ID
	})
	if offset > len(items) {
		return []*domainuser.UserSearchResult{}
	}
	items = items[offset:]
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	out := make([]*domainuser.UserSearchResult, 0, len(items))
	for _, rec := range items {
		out = append(out, &domainuser.UserSearchResult{
			ID:          rec.user.ID,
			Name:        rec.user.Name,
			Avatar:      rec.user.Avatar,
			PublicID:    rec.publicID,
			AccountName: rec.accountName,
		})
	}
	return out
}

func cloneBDDUser(u *domainuser.User) *domainuser.User {
	copied := *u
	copied.RoleCodes = append([]role.Code(nil), u.RoleCodes...)
	return &copied
}

// ─── Friendship repo ─────────────────────────────────────────────────────────

type bddFriendshipRepo struct {
	mu      sync.Mutex
	nextID  int64
	records map[int64]*domainfriendship.Friendship
}

func newBDDFriendshipRepo() *bddFriendshipRepo {
	return &bddFriendshipRepo{}
}

func (r *bddFriendshipRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID = 1000
	r.records = map[int64]*domainfriendship.Friendship{}
}

func (r *bddFriendshipRepo) seed(userID, friendID shared.UserID, status domainfriendship.Status) *domainfriendship.Friendship {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	row := &domainfriendship.Friendship{
		ID:        r.nextID,
		UserID:    userID,
		FriendID:  friendID,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.records[row.ID] = cloneBDDFriendship(row)
	return cloneBDDFriendship(row)
}

func (r *bddFriendshipRepo) FindByUserID(_ context.Context, userID shared.UserID) ([]*domainfriendship.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domainfriendship.Friendship, 0)
	for _, row := range r.records {
		if row.UserID == userID {
			out = append(out, cloneBDDFriendship(row))
		}
	}
	sortFriendships(out)
	return out, nil
}

func (r *bddFriendshipRepo) FindAllByUserID(_ context.Context, userID shared.UserID) ([]*domainfriendship.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domainfriendship.Friendship, 0)
	for _, row := range r.records {
		if row.UserID == userID || row.FriendID == userID {
			out = append(out, cloneBDDFriendship(row))
		}
	}
	sortFriendships(out)
	return out, nil
}

func (r *bddFriendshipRepo) FindPendingByUserID(_ context.Context, userID shared.UserID) ([]*domainfriendship.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.findBy(func(row *domainfriendship.Friendship) bool {
		return row.UserID == userID && row.Status == domainfriendship.StatusPending
	}), nil
}

func (r *bddFriendshipRepo) Create(_ context.Context, userID, friendID shared.UserID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.findPairLocked(userID, friendID); ok {
		return fmt.Errorf("duplicate key value violates unique constraint user_friendships_user_id_friend_id_key")
	}
	r.nextID++
	r.records[r.nextID] = &domainfriendship.Friendship{
		ID:        r.nextID,
		UserID:    userID,
		FriendID:  friendID,
		Status:    domainfriendship.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return nil
}

func (r *bddFriendshipRepo) CreateBlocked(_ context.Context, userID, friendID shared.UserID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.findPairLocked(userID, friendID); ok {
		return fmt.Errorf("duplicate key value violates unique constraint user_friendships_user_id_friend_id_key")
	}
	r.nextID++
	r.records[r.nextID] = &domainfriendship.Friendship{
		ID:        r.nextID,
		UserID:    userID,
		FriendID:  friendID,
		Status:    domainfriendship.StatusBlocked,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return nil
}

func (r *bddFriendshipRepo) FindByID(_ context.Context, id int64) (*domainfriendship.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.records[id]
	if !ok {
		return nil, domainfriendship.ErrFriendshipNotFound
	}
	return cloneBDDFriendship(row), nil
}

func (r *bddFriendshipRepo) FindByUserIDAndFriendID(_ context.Context, userID, friendID shared.UserID) (*domainfriendship.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.findPairLocked(userID, friendID)
	if !ok {
		return nil, domainfriendship.ErrFriendshipNotFound
	}
	return cloneBDDFriendship(row), nil
}

func (r *bddFriendshipRepo) FindBetweenUsers(_ context.Context, userID1, userID2 shared.UserID) ([]*domainfriendship.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.findBy(func(row *domainfriendship.Friendship) bool {
		return (row.UserID == userID1 && row.FriendID == userID2) || (row.UserID == userID2 && row.FriendID == userID1)
	})
	if len(out) == 0 {
		return nil, domainfriendship.ErrFriendshipNotFound
	}
	return out, nil
}

func (r *bddFriendshipRepo) FindPendingByFriendID(_ context.Context, friendID shared.UserID) ([]*domainfriendship.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.findBy(func(row *domainfriendship.Friendship) bool {
		return row.FriendID == friendID && row.Status == domainfriendship.StatusPending
	}), nil
}

func (r *bddFriendshipRepo) UpdateStatus(_ context.Context, id int64, status domainfriendship.Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.records[id]
	if !ok {
		return domainfriendship.ErrFriendshipNotFound
	}
	row.Status = status
	row.UpdatedAt = time.Now()
	return nil
}

func (r *bddFriendshipRepo) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, id)
	return nil
}

func (r *bddFriendshipRepo) findPair(userID, friendID shared.UserID) (*domainfriendship.Friendship, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.findPairLocked(userID, friendID)
	if !ok {
		return nil, false
	}
	return cloneBDDFriendship(row), true
}

func (r *bddFriendshipRepo) findPairLocked(userID, friendID shared.UserID) (*domainfriendship.Friendship, bool) {
	for _, row := range r.records {
		if row.UserID == userID && row.FriendID == friendID {
			return row, true
		}
	}
	return nil, false
}

func (r *bddFriendshipRepo) findBy(match func(*domainfriendship.Friendship) bool) []*domainfriendship.Friendship {
	out := make([]*domainfriendship.Friendship, 0)
	for _, row := range r.records {
		if match(row) {
			out = append(out, cloneBDDFriendship(row))
		}
	}
	sortFriendships(out)
	return out
}

func (r *bddFriendshipRepo) statusCounts() (pending, accepted, blocked int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.records {
		switch row.Status {
		case domainfriendship.StatusPending:
			pending++
		case domainfriendship.StatusAccepted:
			accepted++
		case domainfriendship.StatusBlocked:
			blocked++
		}
	}
	return pending, accepted, blocked
}

func sortFriendships(items []*domainfriendship.Friendship) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}

func cloneBDDFriendship(row *domainfriendship.Friendship) *domainfriendship.Friendship {
	copied := *row
	return &copied
}

// ─── Participant repo ─────────────────────────────────────────────────────────

type bddParticipantRepo struct {
	mu     sync.Mutex
	nextID participant.ID
	byID   map[participant.ID]*participant.Participant
	byUser map[shared.UserID]*participant.Participant
}

func newBDDParticipantRepo() *bddParticipantRepo {
	return &bddParticipantRepo{}
}

func (r *bddParticipantRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID = 1
	r.byID = map[participant.ID]*participant.Participant{}
	r.byUser = map[shared.UserID]*participant.Participant{}
}

// seedForUser creates a participant record for a user if one does not already exist.
func (r *bddParticipantRepo) seedForUser(userID shared.UserID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byUser[userID]; ok {
		return
	}
	p := &participant.Participant{
		ID:        r.nextID,
		Type:      participant.UserType,
		UserID:    &userID,
		CreatedAt: time.Now(),
	}
	r.nextID++
	r.byID[p.ID] = p
	r.byUser[userID] = p
}

func (r *bddParticipantRepo) findByUserID(userID shared.UserID) *participant.Participant {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byUser[userID]
	if !ok {
		return nil
	}
	copied := *p
	return &copied
}

func (r *bddParticipantRepo) FindByID(_ context.Context, id participant.ID) (*participant.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byID[id]
	if !ok {
		return nil, participant.ErrNotFound
	}
	copied := *p
	return &copied, nil
}

func (r *bddParticipantRepo) FindByUserID(_ context.Context, userID shared.UserID) (*participant.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byUser[userID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	copied := *p
	return &copied, nil
}

func (r *bddParticipantRepo) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (r *bddParticipantRepo) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (r *bddParticipantRepo) Create(_ context.Context, p *participant.Participant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p.ID == 0 {
		p.ID = r.nextID
		r.nextID++
	}
	r.byID[p.ID] = p
	if p.UserID != nil {
		r.byUser[*p.UserID] = p
	}
	return nil
}

// ─── ChatRoom repo ────────────────────────────────────────────────────────────

type bddChatRoomRepo struct {
	mu         sync.Mutex
	nextID     chatroom.ID
	byID       map[chatroom.ID]*chatroom.ChatRoom
	chatMember *bddChatMemberRepo
}

func newBDDChatRoomRepo(chatMember *bddChatMemberRepo) *bddChatRoomRepo {
	return &bddChatRoomRepo{chatMember: chatMember}
}

func (r *bddChatRoomRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID = 1
	r.byID = map[chatroom.ID]*chatroom.ChatRoom{}
}

func (r *bddChatRoomRepo) Create(_ context.Context, room *chatroom.ChatRoom) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if room.ID == 0 {
		room.ID = r.nextID
		r.nextID++
	}
	copied := *room
	r.byID[room.ID] = &copied
	return nil
}

func (r *bddChatRoomRepo) FindByID(_ context.Context, id chatroom.ID) (*chatroom.ChatRoom, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	room, ok := r.byID[id]
	if !ok {
		return nil, chatroom.ErrNotFound
	}
	copied := *room
	return &copied, nil
}

func (r *bddChatRoomRepo) FindDirectByParticipants(_ context.Context, p1, p2 participant.ID) (*chatroom.ChatRoom, error) {
	if room, ok := r.findDirect(p1, p2); ok {
		return room, nil
	}
	return nil, chatroom.ErrNotFound
}

func (r *bddChatRoomRepo) Update(_ context.Context, room *chatroom.ChatRoom) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[room.ID]; !ok {
		return chatroom.ErrNotFound
	}
	copied := *room
	r.byID[room.ID] = &copied
	return nil
}

func (r *bddChatRoomRepo) SoftDelete(_ context.Context, id chatroom.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byID, id)
	return nil
}

func (r *bddChatRoomRepo) findDirect(p1, p2 participant.ID) (*chatroom.ChatRoom, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var selected *chatroom.ChatRoom
	for _, room := range r.byID {
		if !r.isDirectRoomForParticipants(room, p1, p2) {
			continue
		}
		if selected == nil || room.ID < selected.ID {
			copied := *room
			selected = &copied
		}
	}
	return selected, selected != nil
}

func (r *bddChatRoomRepo) countDirectRooms(p1, p2 participant.ID) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, room := range r.byID {
		if r.isDirectRoomForParticipants(room, p1, p2) {
			count++
		}
	}
	return count
}

func (r *bddChatRoomRepo) isDirectRoomForParticipants(room *chatroom.ChatRoom, p1, p2 participant.ID) bool {
	if room.Type != chatroom.Direct {
		return false
	}

	members := r.chatMember.findByRoom(room.ID)
	if len(members) != 2 {
		return false
	}

	hasP1 := false
	hasP2 := false
	for _, member := range members {
		if member.ParticipantID == p1 {
			hasP1 = true
		}
		if member.ParticipantID == p2 {
			hasP2 = true
		}
	}

	return hasP1 && hasP2
}

// ─── ChatMember repo ──────────────────────────────────────────────────────────

type bddChatMemberRepo struct {
	mu     sync.Mutex
	nextID chatmember.ID
	byID   map[chatmember.ID]*chatmember.ChatMember
}

func newBDDChatMemberRepo() *bddChatMemberRepo {
	return &bddChatMemberRepo{}
}

func (r *bddChatMemberRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID = 1
	r.byID = map[chatmember.ID]*chatmember.ChatMember{}
}

func (r *bddChatMemberRepo) Add(_ context.Context, m *chatmember.ChatMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m.ID == 0 {
		m.ID = r.nextID
		r.nextID++
	}
	copied := *m
	r.byID[m.ID] = &copied
	return nil
}

func (r *bddChatMemberRepo) FindByID(_ context.Context, id chatmember.ID) (*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.byID[id]
	if !ok {
		return nil, chatmember.ErrNotFound
	}
	copied := *m
	return &copied, nil
}

func (r *bddChatMemberRepo) FindByRoomAndParticipant(_ context.Context, roomID chatroom.ID, participantID participant.ID) (*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.byID {
		if m.RoomID == roomID && m.ParticipantID == participantID && !m.IsDeleted {
			copied := *m
			return &copied, nil
		}
	}
	return nil, chatmember.ErrNotFound
}

func (r *bddChatMemberRepo) FindByRoom(_ context.Context, roomID chatroom.ID) ([]*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.findByRoomLocked(roomID), nil
}

func (r *bddChatMemberRepo) FindByParticipant(_ context.Context, participantID participant.ID) ([]*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*chatmember.ChatMember, 0)
	for _, m := range r.byID {
		if m.ParticipantID == participantID && !m.IsDeleted {
			copied := *m
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *bddChatMemberRepo) Update(_ context.Context, m *chatmember.ChatMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[m.ID]; !ok {
		return chatmember.ErrNotFound
	}
	copied := *m
	r.byID[m.ID] = &copied
	return nil
}

func (r *bddChatMemberRepo) SoftDelete(_ context.Context, id chatmember.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.byID[id]
	if !ok {
		return chatmember.ErrNotFound
	}
	m.IsDeleted = true
	return nil
}

func (r *bddChatMemberRepo) Remove(_ context.Context, roomID chatroom.ID, participantID participant.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, m := range r.byID {
		if m.RoomID == roomID && m.ParticipantID == participantID {
			delete(r.byID, id)
			return nil
		}
	}
	return nil
}

func (r *bddChatMemberRepo) findByRoom(roomID chatroom.ID) []*chatmember.ChatMember {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.findByRoomLocked(roomID)
}

func (r *bddChatMemberRepo) findByRoomLocked(roomID chatroom.ID) []*chatmember.ChatMember {
	out := make([]*chatmember.ChatMember, 0)
	for _, m := range r.byID {
		if m.RoomID == roomID && !m.IsDeleted {
			copied := *m
			out = append(out, &copied)
		}
	}
	return out
}

type bddChatInvitationRepo struct {
	mu     sync.Mutex
	nextID chatinvitation.ID
	byID   map[chatinvitation.ID]*chatinvitation.ChatInvitation
}

func newBDDChatInvitationRepo() *bddChatInvitationRepo {
	return &bddChatInvitationRepo{}
}

func (r *bddChatInvitationRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID = 1
	r.byID = map[chatinvitation.ID]*chatinvitation.ChatInvitation{}
}

func (r *bddChatInvitationRepo) Create(_ context.Context, inv *chatinvitation.ChatInvitation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if inv.ID == 0 {
		inv.ID = r.nextID
		r.nextID++
	}
	copied := *inv
	r.byID[inv.ID] = &copied
	return nil
}

func (r *bddChatInvitationRepo) FindByID(_ context.Context, id chatinvitation.ID) (*chatinvitation.ChatInvitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.byID[id]
	if !ok {
		return nil, chatinvitation.ErrNotFound
	}
	copied := *inv
	return &copied, nil
}

func (r *bddChatInvitationRepo) FindByRoomAndInvitee(_ context.Context, roomID chatroom.ID, inviteeID participant.ID) (*chatinvitation.ChatInvitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inv := range r.byID {
		if inv.RoomID == roomID && inv.InviteeID == inviteeID {
			copied := *inv
			return &copied, nil
		}
	}
	return nil, chatinvitation.ErrNotFound
}

func (r *bddChatInvitationRepo) FindPendingByRoomAndInviter(_ context.Context, roomID chatroom.ID, inviterID participant.ID) (*chatinvitation.ChatInvitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inv := range r.byID {
		if inv.RoomID == roomID && inv.InviterID == inviterID && inv.Status == chatinvitation.Pending {
			copied := *inv
			return &copied, nil
		}
	}
	return nil, chatinvitation.ErrNotFound
}

func (r *bddChatInvitationRepo) FindByRoom(_ context.Context, roomID chatroom.ID) ([]*chatinvitation.ChatInvitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*chatinvitation.ChatInvitation, 0)
	for _, inv := range r.byID {
		if inv.RoomID == roomID {
			copied := *inv
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *bddChatInvitationRepo) UpdateStatus(_ context.Context, id chatinvitation.ID, status chatinvitation.Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.byID[id]
	if !ok {
		return chatinvitation.ErrNotFound
	}
	inv.Status = status
	return nil
}

var (
	_ domainuser.Repository       = (*bddUserRepo)(nil)
	_ domainfriendship.Repository = (*bddFriendshipRepo)(nil)
	_ participant.Repository      = (*bddParticipantRepo)(nil)
	_ chatroom.Repository         = (*bddChatRoomRepo)(nil)
	_ chatmember.Repository       = (*bddChatMemberRepo)(nil)
	_ chatinvitation.Repository   = (*bddChatInvitationRepo)(nil)
)
