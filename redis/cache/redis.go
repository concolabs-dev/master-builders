package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	Ctx = context.Background()
	RDB *redis.Client
)

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // Set if Redis has password
		DB:       0,
	})
}

func SetCache(key string, value string, ttl time.Duration) error {
	return RDB.Set(Ctx, key, value, ttl).Err()
}

func GetCache(key string) (string, error) {
	return RDB.Get(Ctx, key).Result()
}
