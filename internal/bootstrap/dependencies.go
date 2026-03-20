package bootstrap

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appcrypto "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/crypto"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	appPort "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/port"
	appPush "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/push"
	appSecurity "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	domainAgent "github.com/HiroLiang/tentserv-chat-server/internal/domain/agent"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/cache"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/auth/session"
	infraVerification "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/auth/verification"
	infraEmail "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/database"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	postgresAccount "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/account"
	postgresAgent "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/agent"
	postgresChat "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/chat"
	postgresDeliveryQueue "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/deliveryqueue"
	postgresDevice "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/device"
	postgresE2EE "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/e2ee"
	postgresEmail "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/email"
	postgresFriendship "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/friendship"
	postgresSession "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/session"
	postgresUser "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/user"
	postgresUserRole "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/userrole"
	infraPush "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/push"
	infraRedis "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/redis"
	infraRedisSecurity "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/redis/security"
	infraCrypto "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/shared/crypto"
	infraSharedSecurity "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/shared/security"
	infraStorage "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/shared/storage"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/ws"
	"github.com/redis/go-redis/v9"
)

// Dependencies Base dependency container
type Dependencies struct {
	Uow            transaction.UnitOfWork
	SessionManager port.SessionManager
	RateLimiter    appSecurity.RateLimiter
	RedisCache     cache.Cache

	PwdHasher     appSecurity.Hasher
	ContextHasher appSecurity.Hasher

	LocalFileStorage appPort.FileStorage

	HMacer appSecurity.HMACer

	VerificationStore port.VerificationStore
	EmailService      appEmail.EmailService

	AccountRepo           account.Repository
	UserRepo              user.Repository
	UserRoleRepo          userrole.Repository
	DeviceRepo            device.Repository
	ParticipantRepository participant.Repository
	ChatRoomRepo          chatroom.Repository
	ChatMemberRepo        chatmember.Repository
	ChatInvitationRepo    chatinvitation.Repository
	ChatMessageRepo       chatmessage.Repository
	AgentRepo             domainAgent.Repository

	KeyVerifier         appcrypto.KeyVerifier
	IdentityKeyRepo     useridentitykey.Repository
	SignedPreKeyRepo    usersignedprekey.Repository
	OTPPreKeyRepo       userotpprekey.Repository
	MemberSenderKeyRepo membersenderkey.Repository

	Hub               *ws.Hub
	DeliveryQueueRepo deliveryqueue.Repository
	PushDispatcher    appPush.Dispatcher

	FriendshipRepo friendship.Repository
}

func BuildDeps(redis *redis.Client, dataSources *database.DataSources) (*Dependencies, error) {

	// get config
	conf := config.App()

	// Postgres datasource
	postgresDB := dataSources.GetDB(database.Postgres)

	// Redis cache
	redisCache := infraRedis.NewRedisCache(redis)

	// Rate limiter
	rateLimitRepo := infraRedisSecurity.NewRedisRateLimitRepository(redis)
	rateLimiter := infraSharedSecurity.NewRedisRateLimiter(
		rateLimitRepo,
		security.RateLimitPolicy{
			Limit:  conf.RateLimitConfig.GlobalLimit,
			Window: conf.RateLimitConfig.GlobalUnit,
		},
		security.RateLimitPolicy{
			Limit:  conf.RateLimitConfig.IPLimit,
			Window: conf.RateLimitConfig.IPUnit,
		},
	)

	sessionRepo := postgresSession.NewSessionRepository(postgresDB)
	emailRecorder := postgresEmail.NewPostgresEmailRecorder(postgresDB)

	// Build Hub + delivery queue + push dispatcher
	hub := ws.NewHub()
	go hub.Run()
	deliveryQueueRepo := postgresDeliveryQueue.NewDeliveryQueueRepository(postgresDB)
	pushDispatcher := infraPush.NewDBDispatcher(deliveryQueueRepo, hub)

	return &Dependencies{
		Uow: postgres.NewPostgresUnitOfWork(postgresDB),
		SessionManager: session.NewSessionManager(
			redisCache,
			sessionRepo,
			conf.AuthToken.Expiration,
		),
		RateLimiter:           rateLimiter,
		RedisCache:            redisCache,
		PwdHasher:             infraSharedSecurity.NewArgon2Hasher(),
		ContextHasher:         infraSharedSecurity.NewContentHasher(),
		LocalFileStorage:      infraStorage.NewLocalFileStorage(conf.Storage.BasePath, conf.Storage.BaseURL),
		HMacer:                infraSharedSecurity.NewSHA256HMACer(conf.Secrets.HmacSecret),
		VerificationStore:     infraVerification.NewVerificationStore(redisCache),
		EmailService:          infraEmail.NewResendEmailService(conf.Email.ApiKey, emailRecorder),
		AccountRepo:           postgresAccount.NewAccountRepo(postgresDB),
		UserRepo:              postgresUser.NewUserRepository(postgresDB),
		UserRoleRepo:          postgresUserRole.NewUserRoleRepository(postgresDB),
		DeviceRepo:            postgresDevice.NewDeviceRepository(postgresDB),
		ParticipantRepository: postgresChat.NewParticipantRepository(postgresDB),
		ChatRoomRepo:          postgresChat.NewChatRoomRepository(postgresDB),
		ChatMemberRepo:        postgresChat.NewChatMemberRepository(postgresDB),
		ChatInvitationRepo:    postgresChat.NewChatInvitationRepository(postgresDB),
		ChatMessageRepo:       postgresChat.NewChatMessageRepository(postgresDB),
		AgentRepo:             postgresAgent.NewAgentRepository(postgresDB),
		KeyVerifier:           infraCrypto.NewE2EEVerifier(),
		IdentityKeyRepo:       postgresE2EE.NewIdentityKeyRepository(postgresDB),
		SignedPreKeyRepo:      postgresE2EE.NewSignedPreKeyRepository(postgresDB),
		OTPPreKeyRepo:         postgresE2EE.NewOTPPreKeyRepository(postgresDB),
		MemberSenderKeyRepo:   postgresE2EE.NewSenderKeyRepository(postgresDB),
		Hub:                   hub,
		DeliveryQueueRepo:     deliveryQueueRepo,
		PushDispatcher:        pushDispatcher,
		FriendshipRepo:        postgresFriendship.NewFriendshipRepository(postgresDB),
	}, nil
}

func MockDeps(opts ...DepsOption) *Dependencies {

	//Default dependencies
	deps := &Dependencies{}

	// Optionals
	for _, opt := range opts {
		opt(deps)
	}

	return deps
}

type DepsOption func(*Dependencies)
