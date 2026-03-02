package bootstrap

import (
	"github.com/HiroLiang/goat-server/internal/application/shared/auth"
	"github.com/HiroLiang/goat-server/internal/application/shared/security"
	"github.com/HiroLiang/goat-server/internal/application/shared/storage"
	"github.com/HiroLiang/goat-server/internal/config"
	"github.com/HiroLiang/goat-server/internal/domain/agent"
	"github.com/HiroLiang/goat-server/internal/domain/chatgroup"
	"github.com/HiroLiang/goat-server/internal/domain/chatmember"
	"github.com/HiroLiang/goat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/goat-server/internal/domain/device"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
	domainSecurity "github.com/HiroLiang/goat-server/internal/domain/security"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/domain/userrole"
	"github.com/HiroLiang/goat-server/internal/infrastructure/auth/session"
	infraAuth "github.com/HiroLiang/goat-server/internal/infrastructure/auth/token"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/database"
	dbAgent "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/agent"
	dbChat "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/chat"
	dbDevice "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/device"
	dbUser "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/user"
	dbUserrole "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/userrole"
	redisInfra "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/redis"
	redisInfraSecurity "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/redis/security"
	redisUserrole "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/redis/userrole"
	infraSecurity "github.com/HiroLiang/goat-server/internal/infrastructure/shared/security"
	infraStorage "github.com/HiroLiang/goat-server/internal/infrastructure/storage"
	"github.com/redis/go-redis/v9"
)

// Dependencies Base dependency container
type Dependencies struct {
	AgentRepo       agent.Repository
	ChatGroupRepo   chatgroup.Repository
	ChatMemberRepo  chatmember.Repository
	ChatMessageRepo chatmessage.Repository
	DeviceRepo      device.Repository
	FileStorage     storage.FileStorage
	Argon2Hasher    *infraSecurity.Argon2Hasher
	ContextHasher   *infraSecurity.ContentHasher
	HMACer          security.HMACer
	ParticipantRepo participant.Repository
	RateLimiter     security.RateLimiter
	TokenService    auth.TokenService
	UserRepo        user.Repository
	UserRoleRepo    userrole.Repository
}

func BuildDeps(redis *redis.Client, dataSources *database.DataSources) (*Dependencies, error) {

	// get config
	conf := config.App()

	// Postgres datasource
	postgres := dataSources.GetDB(database.Postgres)

	// redis cache
	redisCache := redisInfra.NewRedisCache(redis)

	// Session store
	sessionStore := session.NewRedisSessionStore(redisCache, redis)

	// Content hasher
	contextHasher := infraSecurity.NewContentHasher()

	return &Dependencies{
		AgentRepo:       dbAgent.NewAgentRepository(postgres),
		ChatGroupRepo:   dbChat.NewChatGroupRepository(postgres),
		ChatMemberRepo:  dbChat.NewChatMemberRepository(postgres),
		ChatMessageRepo: dbChat.NewChatMessageRepository(postgres),
		DeviceRepo:      dbDevice.NewDeviceRepository(postgres),
		FileStorage:     infraStorage.NewLocalFileStorage(conf.Storage.LocalPath, contextHasher),
		Argon2Hasher:    infraSecurity.NewArgon2Hasher(),
		ContextHasher:   contextHasher,
		HMACer:          infraSecurity.NewSHA256HMACer(conf.Secrets.HmacSecret),
		ParticipantRepo: dbChat.NewParticipantRepository(postgres),
		RateLimiter:     buildRateLimiter(redis, conf),
		TokenService:    infraAuth.NewAuthTokenService(sessionStore, conf.AuthToken.Expiration),
		UserRepo:        dbUser.NewUserRepository(postgres),
		UserRoleRepo:    redisUserrole.NewUserRoleCachedRepo(redisCache, dbUserrole.NewUserRoleRepository(postgres)),
	}, nil
}

func MockDeps(opts ...DepsOption) *Dependencies {
	conf := config.App()

	//Default dependencies
	deps := &Dependencies{
		Argon2Hasher: infraSecurity.NewArgon2Hasher(),
		HMACer:       infraSecurity.NewSHA256HMACer(conf.Secrets.HmacSecret),
	}

	// Optionals
	for _, opt := range opts {
		opt(deps)
	}

	return deps
}

type DepsOption func(*Dependencies)

// buildRateLimiter build rate limiter
func buildRateLimiter(redis *redis.Client, conf *config.AppConfig) security.RateLimiter {
	rateLimitConf := conf.RateLimitConfig
	globalPolicy := domainSecurity.RateLimitPolicy{
		Limit:  int64(rateLimitConf.GlobalLimit),
		Window: rateLimitConf.GlobalUnit,
	}
	ipPolicy := domainSecurity.RateLimitPolicy{
		Limit:  int64(rateLimitConf.IPLimit),
		Window: rateLimitConf.IPUnit,
	}
	return infraSecurity.NewRedisRateLimiter(
		redisInfraSecurity.NewRedisRateLimitRepository(redis),
		globalPolicy,
		ipPolicy)
}
