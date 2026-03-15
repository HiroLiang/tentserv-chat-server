package bootstrap

import (
	"github.com/HiroLiang/goat-server/internal/application/auth/port"
	appEmail "github.com/HiroLiang/goat-server/internal/application/shared/email"
	appPort "github.com/HiroLiang/goat-server/internal/application/shared/port"
	appSecurity "github.com/HiroLiang/goat-server/internal/application/shared/security"
	"github.com/HiroLiang/goat-server/internal/config"
	"github.com/HiroLiang/goat-server/internal/domain/account"
	"github.com/HiroLiang/goat-server/internal/domain/cache"
	"github.com/HiroLiang/goat-server/internal/domain/device"
	"github.com/HiroLiang/goat-server/internal/domain/security"
	"github.com/HiroLiang/goat-server/internal/domain/transaction"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/infrastructure/auth/session"
	infraVerification "github.com/HiroLiang/goat-server/internal/infrastructure/auth/verification"
	infraEmail "github.com/HiroLiang/goat-server/internal/infrastructure/email"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/database"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	postgresAccount "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/account"
	postgresDevice "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/device"
	postgresEmail "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/email"
	postgresSession "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/session"
	postgresUser "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/user"
	infraRedis "github.com/HiroLiang/goat-server/internal/infrastructure/redis"
	infraRedisSecurity "github.com/HiroLiang/goat-server/internal/infrastructure/redis/security"
	infraSharedSecurity "github.com/HiroLiang/goat-server/internal/infrastructure/shared/security"
	infraStorage "github.com/HiroLiang/goat-server/internal/infrastructure/shared/storage"
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

	AccountRepo account.Repository
	UserRepo    user.Repository
	DeviceRepo  device.Repository
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

	return &Dependencies{
		Uow: postgres.NewPostgresUnitOfWork(postgresDB),
		SessionManager: session.NewSessionManager(
			redisCache,
			sessionRepo,
			conf.AuthToken.Expiration,
		),
		RateLimiter:       rateLimiter,
		RedisCache:        redisCache,
		PwdHasher:         infraSharedSecurity.NewArgon2Hasher(),
		ContextHasher:     infraSharedSecurity.NewContentHasher(),
		LocalFileStorage:  infraStorage.NewLocalFileStorage(),
		HMacer:            infraSharedSecurity.NewSHA256HMACer(conf.Secrets.HmacSecret),
		VerificationStore: infraVerification.NewVerificationStore(redisCache),
		EmailService:      infraEmail.NewResendEmailService(conf.Email.ApiKey, emailRecorder),
		AccountRepo:       postgresAccount.NewAccountRepo(postgresDB),
		UserRepo:          postgresUser.NewUserRepository(postgresDB),
		DeviceRepo:        postgresDevice.NewDeviceRepository(postgresDB),
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
