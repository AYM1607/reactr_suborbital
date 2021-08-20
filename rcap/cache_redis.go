package rcap

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

type RedisCache struct {
	config CacheConfig
	client *redis.Client
}

type RedisConfig struct {
	Addr     string `json:"address" yaml:"address"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

func newRedisCache(config CacheConfig) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisConfig.Addr,
		Username: config.RedisConfig.Username,
		Password: config.RedisConfig.Password,
	})

	rc := &RedisCache{
		config: config,
		client: client,
	}

	return rc
}

// Set sets a value in the cache
func (r *RedisCache) Set(key string, val []byte, ttl int) error {
	if !r.config.Enabled {
		return ErrCapabilityNotEnabled
	}

	ttlDuration := time.Duration(time.Second * time.Duration(ttl))

	if err := r.client.Set(context.Background(), key, val, ttlDuration).Err(); err != nil {
		return errors.Wrap(err, "failed to client.Set")
	}

	return nil
}

// Get gets a value from the cache
func (r *RedisCache) Get(key string) ([]byte, error) {
	if !r.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	}

	val, err := r.client.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to client.Get")
	}

	return val, nil
}

// Delete deletes a key
func (r *RedisCache) Delete(key string) error {
	if !r.config.Enabled {
		return ErrCapabilityNotEnabled
	}

	if _, err := r.client.Del(context.Background(), key).Result(); err != nil {
		return errors.Wrap(err, "failed to client.Del")
	}

	return nil
}
