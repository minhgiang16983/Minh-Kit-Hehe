package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const DefaultCommandTimeout = 5 * time.Second

type RedisStoreInterface interface {
	GetClient() *redis.Client
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Expire(ctx context.Context, key string, expiration time.Duration) error
	Exists(ctx context.Context, keys ...string) (bool, error)
	Incr(ctx context.Context, key string) (int64, error)
}

func (s *RedisStore) GetClient() *redis.Client {
	return s.Client
}

func (r *RedisStore) Get(ctx context.Context, key string) (string, error) {
	ctx, cancel := r.wrapperCtx(ctx)
	defer cancel()

	val, err := r.Client.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", err
	}

	return val, nil
}

func (r *RedisStore) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	ctx, cancel := r.wrapperCtx(ctx)
	defer cancel()

	err := r.Client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisStore) Del(ctx context.Context, keys ...string) error {
	ctx, cancel := r.wrapperCtx(ctx)
	defer cancel()

	return r.Client.Del(ctx, keys...).Err()
}

func (r *RedisStore) Exists(ctx context.Context, keys ...string) (bool, error) {
	ctx, cancel := r.wrapperCtx(ctx)
	defer cancel()

	val, err := r.Client.Exists(ctx, keys...).Result()
	if err != nil {
		return false, err
	}

	return val == int64(len(keys)), nil
}

func (r *RedisStore) Expire(ctx context.Context, key string, expiration time.Duration) error {
	ctx, cancel := r.wrapperCtx(ctx)
	defer cancel()

	return r.Client.Expire(ctx, key, expiration).Err()
}

func (r *RedisStore) Incr(ctx context.Context, key string) (int64, error) {
	ctx, cancel := r.wrapperCtx(ctx)
	defer cancel()
	val, err := r.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (r *RedisStore) wrapperCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); !ok {
		return context.WithTimeout(ctx, DefaultCommandTimeout)
	}
	return ctx, func() {}
}
