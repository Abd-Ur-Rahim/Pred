package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client    *redis.Client
	pubKeyTTL time.Duration
	nonceTTL  time.Duration
}

func NewRedisCache(addr, password string, db int, pubKeyTTL, nonceTTL time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisCache{
		client:    client,
		pubKeyTTL: pubKeyTTL,
		nonceTTL:  nonceTTL,
	}, nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func (r *RedisCache) CacheDevicePublicKey(ctx context.Context, deviceID uint, pemKey string) error {
	key := fmt.Sprintf("device_pubkey:%d", deviceID)
	return r.client.Set(ctx, key, pemKey, r.pubKeyTTL).Err()
}

func (r *RedisCache) GetDevicePublicKey(ctx context.Context, deviceID uint) (string, error) {
	key := fmt.Sprintf("device_pubkey:%d", deviceID)
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCache) CacheDeviceState(ctx context.Context, deviceID uint, isActive bool, publicKey string) error {
	key := fmt.Sprintf("device_state:%d", deviceID)
	if err := r.client.HSet(ctx, key, map[string]interface{}{
		"is_active":  strconv.FormatBool(isActive),
		"public_key": publicKey,
	}).Err(); err != nil {
		return err
	}

	return r.client.Expire(ctx, key, r.pubKeyTTL).Err()
}

func (r *RedisCache) GetDeviceState(ctx context.Context, deviceID uint) (bool, string, bool, error) {
	key := fmt.Sprintf("device_state:%d", deviceID)
	values, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return false, "", false, err
	}
	if len(values) == 0 {
		return false, "", false, nil
	}

	isActive, err := strconv.ParseBool(values["is_active"])
	if err != nil {
		return false, "", false, err
	}

	return isActive, values["public_key"], true, nil
}

func (r *RedisCache) UpdateDeviceActiveStatus(ctx context.Context, deviceID uint, isActive bool) error {
	key := fmt.Sprintf("device_state:%d", deviceID)
	if err := r.client.HSet(ctx, key, "is_active", strconv.FormatBool(isActive)).Err(); err != nil {
		return err
	}

	return r.client.Expire(ctx, key, r.pubKeyTTL).Err()
}

func (r *RedisCache) ReserveNonce(ctx context.Context, deviceID uint, nonce string) (bool, error) {
	key := fmt.Sprintf("nonce:%d:%s", deviceID, nonce)
	return r.client.SetNX(ctx, key, "true", r.nonceTTL).Result()
}

func (r *RedisCache) NonceExists(ctx context.Context, deviceID uint, nonce string) (bool, error) {
	key := fmt.Sprintf("nonce:%d:%s", deviceID, nonce)
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *RedisCache) MarkNonceUsed(ctx context.Context, deviceID uint, nonce string) (bool, error) {
	key := fmt.Sprintf("nonce:%d:%s", deviceID, nonce)
	return r.client.SetNX(ctx, key, "true", r.nonceTTL).Result()
}
