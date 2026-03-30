package bootstrap

import (
	"time"

	authUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/usecase"
	chatUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/usecase"
	deviceUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/device/usecase"
	e2eeUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/usecase"
	friendshipUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/friendship/usecase"
	ollamaUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/ollama/usecase"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	userUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/user/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	infraBuilder "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/email/builder"
)

type UseCases struct {
	RegisterUseCase          *authUseCase.RegisterUseCase
	LoginUseCase             *authUseCase.LoginUseCase
	LogoutUseCase            *authUseCase.LogoutUseCase
	GetAccountProfileUseCase *authUseCase.GetProfileUseCase
	VerifyEmailUseCase       *authUseCase.VerifyEmailUseCase

	UpdateUserProfileUseCase *userUseCase.UpdateProfileUseCase
	UploadAvatarUseCase      *userUseCase.UploadAvatarUseCase
	GetUserProfileUseCase    *userUseCase.GetProfileUseCase
	SearchUsersUseCase       *userUseCase.SearchUsersUseCase

	RegisterDeviceUseCase   *deviceUseCase.RegisterUseCase
	GetDeviceProfileUseCase *deviceUseCase.GetProfileUseCase
	UpdateDeviceUseCase     *deviceUseCase.UpdateDeviceUseCase
	ListDevicesUseCase      *deviceUseCase.ListDevicesUseCase
	BindAccountUseCase      *deviceUseCase.BindAccountUseCase
	DeleteDeviceUseCase     *deviceUseCase.DeleteDeviceUseCase

	CreateUserParticipantUseCase *chatUseCase.CreateUserParticipantUseCase
	GetUserParticipantUseCase    *chatUseCase.GetUserParticipantUseCase

	CreateChatRoomUseCase      *chatUseCase.CreateChatRoomUseCase
	JoinChatRoomUseCase        *chatUseCase.JoinChatRoomUseCase
	ApproveJoinRequestUseCase  *chatUseCase.ApproveJoinRequestUseCase
	GetMyRoomInvitationUseCase *chatUseCase.GetMyRoomInvitationUseCase
	RespondToInvitationUseCase *chatUseCase.RespondToInvitationUseCase
	GetUserChatRoomsUseCase    *chatUseCase.GetUserChatRoomsUseCase
	GetChatRoomDetailUseCase   *chatUseCase.GetChatRoomDetailUseCase
	GetChatRoomMessagesUseCase *chatUseCase.GetChatRoomMessagesUseCase
	UpdateMemberStatusUseCase  *chatUseCase.UpdateMemberStatusUseCase

	UploadIdentityKeyUseCase                  *e2eeUseCase.UploadIdentityKeyUseCase
	UploadSignedPreKeyUseCase                 *e2eeUseCase.UploadSignedPreKeyUseCase
	UploadOTPPreKeysUseCase                   *e2eeUseCase.UploadOTPPreKeysUseCase
	CountOTPPreKeysUseCase                    *e2eeUseCase.CountOTPPreKeysUseCase
	GetKeyBundleUseCase                       *e2eeUseCase.GetKeyBundleUseCase
	UploadSenderKeyUseCase                    *e2eeUseCase.UploadSenderKeyUseCase
	GetSenderKeysUseCase                      *e2eeUseCase.GetSenderKeysUseCase
	GetSenderKeyDistributionStatusUseCase     *e2eeUseCase.GetSenderKeyDistributionStatusUseCase

	SendMessageUseCase     *chatUseCase.SendMessageUseCase
	UploadRoomMediaUseCase *chatUseCase.UploadRoomMediaUseCase

	GetFriendsUseCase        *friendshipUseCase.GetFriendsUseCase
	ApplyFriendshipUseCase   *friendshipUseCase.ApplyFriendshipUseCase
	AcceptFriendshipUseCase  *friendshipUseCase.AcceptFriendshipUseCase
	GetFriendRequestsUseCase *friendshipUseCase.GetFriendRequestsUseCase
	RemoveFriendshipUseCase  *friendshipUseCase.RemoveFriendshipUseCase
	GetSentRequestsUseCase   *friendshipUseCase.GetSentRequestsUseCase
	CancelSentRequestUseCase *friendshipUseCase.CancelSentRequestUseCase
	BlockUserUseCase         *friendshipUseCase.BlockUserUseCase
	UnblockUserUseCase       *friendshipUseCase.UnblockUserUseCase

	StreamChatUseCase *ollamaUseCase.StreamChatUseCase
}

func BuildUseCases(deps *Dependencies) *UseCases {
	conf := config.App()
	sender := shared.EmailSender{
		Address: shared.EmailAddress(conf.Email.SenderAddress),
		Name:    conf.Email.SenderName,
	}

	return &UseCases{
		RegisterUseCase: authUseCase.NewRegisterUseCase(
			deps.Uow,
			deps.PwdHasher,
			deps.AccountRepo,
			deps.UserRepo,
			deps.UserRoleRepo,
			deps.VerificationStore,
			deps.EmailService,
			func(recipientEmail, recipientName, verifyURL string) appEmail.EmailBuilder {
				return infraBuilder.NewRegisterMailBuilder(sender, recipientEmail, recipientName, verifyURL)
			},
		),
		LoginUseCase: authUseCase.NewLoginUseCase(
			deps.Uow, deps.PwdHasher, deps.SessionManager,
			deps.AccountRepo, deps.UserRepo, deps.UserRoleRepo,
			deps.EmailService,
			func(recipientEmail, recipientName, deviceID, ip string, loginTime time.Time) appEmail.EmailBuilder {
				return infraBuilder.NewLoginMailBuilder(sender, recipientEmail, recipientName, deviceID, ip, loginTime)
			},
		),
		LogoutUseCase:            authUseCase.NewLogoutUseCase(deps.SessionManager),
		GetAccountProfileUseCase: authUseCase.NewGetProfileUseCase(deps.AccountRepo, deps.UserRepo),
		VerifyEmailUseCase:       authUseCase.NewVerifyEmailUseCase(deps.VerificationStore, deps.AccountRepo),

		UpdateUserProfileUseCase: userUseCase.NewUpdateProfileUseCase(deps.UserRepo, deps.UserRoleRepo),
		UploadAvatarUseCase:      userUseCase.NewUploadAvatarUseCase(deps.ContextHasher, deps.LocalFileStorage, deps.UserRepo),
		GetUserProfileUseCase:    userUseCase.NewGetProfileUseCase(deps.UserRepo),
		SearchUsersUseCase:       userUseCase.NewSearchUsersUseCase(deps.UserRepo, deps.FriendshipRepo),

		RegisterDeviceUseCase:   deviceUseCase.NewRegisterUseCase(deps.Uow, deps.DeviceRepo),
		GetDeviceProfileUseCase: deviceUseCase.NewGetProfileUseCase(deps.Uow, deps.DeviceRepo),
		UpdateDeviceUseCase:     deviceUseCase.NewUpdateDeviceUseCase(deps.DeviceRepo),
		ListDevicesUseCase:      deviceUseCase.NewListDevicesUseCase(deps.DeviceRepo),
		BindAccountUseCase:      deviceUseCase.NewBindAccountUseCase(deps.DeviceRepo),
		DeleteDeviceUseCase:     deviceUseCase.NewDeleteDeviceUseCase(deps.DeviceRepo),

		CreateUserParticipantUseCase: chatUseCase.NewCreateUserParticipantUseCase(deps.Uow, deps.ParticipantRepository),
		GetUserParticipantUseCase:    chatUseCase.NewGetUserParticipantUseCase(deps.ParticipantRepository),

		CreateChatRoomUseCase: chatUseCase.NewCreateChatRoomUseCase(
			deps.Uow,
			deps.ChatRoomRepo,
			deps.ChatMemberRepo,
			deps.ParticipantRepository,
			deps.FriendshipRepo,
			deps.ChatInvitationRepo,
		),
		JoinChatRoomUseCase: chatUseCase.NewJoinChatRoomUseCase(
			deps.Uow,
			deps.ChatRoomRepo,
			deps.ChatMemberRepo,
			deps.ParticipantRepository,
			deps.ChatInvitationRepo,
		),
		ApproveJoinRequestUseCase: chatUseCase.NewApproveJoinRequestUseCase(
			deps.Uow,
			deps.ChatMemberRepo,
			deps.ParticipantRepository,
			deps.ChatInvitationRepo,
		),
		GetMyRoomInvitationUseCase: chatUseCase.NewGetMyRoomInvitationUseCase(
			deps.ParticipantRepository,
			deps.ChatInvitationRepo,
			deps.UserRepo,
		),
		RespondToInvitationUseCase: chatUseCase.NewRespondToInvitationUseCase(
			deps.Uow,
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
			deps.ChatInvitationRepo,
			deps.FriendshipRepo,
		),
		GetUserChatRoomsUseCase: chatUseCase.NewGetUserChatRoomsUseCase(
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
			deps.ChatRoomRepo,
			deps.ChatMessageRepo,
			deps.UserRepo,
			deps.AgentRepo,
		),
		GetChatRoomDetailUseCase: chatUseCase.NewGetChatRoomDetailUseCase(
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
			deps.ChatRoomRepo,
			deps.ChatMessageRepo,
			deps.UserRepo,
			deps.AgentRepo,
		),
		GetChatRoomMessagesUseCase: chatUseCase.NewGetChatRoomMessagesUseCase(
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
			deps.ChatMessageRepo,
		),
		UpdateMemberStatusUseCase: chatUseCase.NewUpdateMemberStatusUseCase(
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
		),

		UploadIdentityKeyUseCase: e2eeUseCase.NewUploadIdentityKeyUseCase(
			deps.IdentityKeyRepo,
			deps.KeyVerifier,
		),
		UploadSignedPreKeyUseCase: e2eeUseCase.NewUploadSignedPreKeyUseCase(
			deps.IdentityKeyRepo,
			deps.SignedPreKeyRepo,
			deps.KeyVerifier,
		),
		UploadOTPPreKeysUseCase: e2eeUseCase.NewUploadOTPPreKeysUseCase(
			deps.OTPPreKeyRepo,
		),
		CountOTPPreKeysUseCase: e2eeUseCase.NewCountOTPPreKeysUseCase(
			deps.OTPPreKeyRepo,
		),
		GetKeyBundleUseCase: e2eeUseCase.NewGetKeyBundleUseCase(
			deps.IdentityKeyRepo,
			deps.SignedPreKeyRepo,
			deps.OTPPreKeyRepo,
			deps.PushDispatcher,
		),
		UploadSenderKeyUseCase: e2eeUseCase.NewUploadSenderKeyUseCase(
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
			deps.MemberSenderKeyRepo,
		),
		GetSenderKeysUseCase: e2eeUseCase.NewGetSenderKeysUseCase(
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
			deps.MemberSenderKeyRepo,
			deps.SenderKeyDistributionRepo,
		),
		GetSenderKeyDistributionStatusUseCase: e2eeUseCase.NewGetSenderKeyDistributionStatusUseCase(
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
			deps.MemberSenderKeyRepo,
			deps.SenderKeyDistributionRepo,
		),

		SendMessageUseCase: chatUseCase.NewSendMessageUseCase(
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
			deps.ChatMessageRepo,
			deps.Hub,
		),
		UploadRoomMediaUseCase: chatUseCase.NewUploadRoomMediaUseCase(
			deps.ParticipantRepository,
			deps.ChatMemberRepo,
			deps.LocalFileStorage,
		),

		GetFriendsUseCase:        friendshipUseCase.NewGetFriendsUseCase(deps.FriendshipRepo, deps.UserRepo),
		ApplyFriendshipUseCase:   friendshipUseCase.NewApplyFriendshipUseCase(deps.FriendshipRepo),
		AcceptFriendshipUseCase: friendshipUseCase.NewAcceptFriendshipUseCase(
			deps.Uow,
			deps.FriendshipRepo,
			deps.ParticipantRepository,
			deps.ChatRoomRepo,
			deps.ChatMemberRepo,
		),
		GetFriendRequestsUseCase:   friendshipUseCase.NewGetFriendRequestsUseCase(deps.FriendshipRepo, deps.UserRepo),
		RemoveFriendshipUseCase:    friendshipUseCase.NewRemoveFriendshipUseCase(deps.FriendshipRepo),
		GetSentRequestsUseCase:     friendshipUseCase.NewGetSentRequestsUseCase(deps.FriendshipRepo, deps.UserRepo),
		CancelSentRequestUseCase:   friendshipUseCase.NewCancelSentRequestUseCase(deps.FriendshipRepo),
		BlockUserUseCase:           friendshipUseCase.NewBlockUserUseCase(deps.FriendshipRepo),
		UnblockUserUseCase:         friendshipUseCase.NewUnblockUserUseCase(deps.FriendshipRepo),

		StreamChatUseCase: ollamaUseCase.NewStreamChatUseCase(deps.RedisCache),
	}
}
