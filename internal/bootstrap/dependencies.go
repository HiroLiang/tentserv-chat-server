package bootstrap

import (
	"github.com/HiroLiang/goat-server/internal/application/auth/port"
	appPort "github.com/HiroLiang/goat-server/internal/application/shared/port"
	appSecurity "github.com/HiroLiang/goat-server/internal/application/shared/security"
	"github.com/HiroLiang/goat-server/internal/config"
	"github.com/HiroLiang/goat-server/internal/domain/account"
	"github.com/HiroLiang/goat-server/internal/domain/cache"
	"github.com/HiroLiang/goat-server/internal/domain/security"
	"github.com/HiroLiang/goat-server/internal/domain/transaction"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/infrastructure/auth/session"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/database"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	postgresAccount "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/account"
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

	AccountRepo account.Repository
	UserRepo    user.Repository
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

	return &Dependencies{
		Uow:              postgres.NewPostgresUnitOfWork(postgresDB),
		SessionManager:   session.NewSessionManager(redisCache),
		RateLimiter:      rateLimiter,
		RedisCache:       redisCache,
		PwdHasher:        infraSharedSecurity.NewArgon2Hasher(),
		ContextHasher:    infraSharedSecurity.NewContentHasher(),
		LocalFileStorage: infraStorage.NewLocalFileStorage(),
		HMacer:           infraSharedSecurity.NewSHA256HMACer(conf.Secrets.HmacSecret),
		AccountRepo:      postgresAccount.NewAccountRepo(postgresDB),
		UserRepo:         postgresUser.NewUserRepository(postgresDB),
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
