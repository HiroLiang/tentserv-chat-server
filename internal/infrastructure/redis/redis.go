package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func InitRedis(config *ClientConfig) (*redis.Client, error) {
	if config.Addr == "" {
		fmt.Println("without using redis")
		return nil, nil
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect Redis: %v", err)
	}

	fmt.Println("Connected to Redis")
	return redisClient, nil
}
