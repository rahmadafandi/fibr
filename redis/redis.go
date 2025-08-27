package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type (
	Client = redis.Client
)

// Redis is a struct that holds the redis client
type Redis struct {
	Client *Client
}

// New is a function to create a new redis client
func New(client *Client) *Redis {
	return &Redis{
		Client: client,
	}
}

// Set is a function to set a value to redis
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	p, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.Client.Set(ctx, key, p, expiration).Err()
}

// Get is a function to get a value from redis
func (r *Redis) Get(ctx context.Context, key string, dest interface{}) error {
	p, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(p, dest)
}

// SetGormResult is a function to set a gorm result to redis
func (r *Redis) SetGormResult(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	p, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.Client.Set(ctx, key, p, expiration).Err()
}

// GetGormResult is a function to get a gorm result from redis
func (r *Redis) GetGormResult(ctx context.Context, key string, dest interface{}) error {
	p, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(p, dest)
}

// GormResult is a struct that holds the gorm result
type GormResult struct {
	DB *gorm.DB
}

// NewGormResult is a function to create a new gorm result
func NewGormResult(db *gorm.DB) *GormResult {
	return &GormResult{
		DB: db,
	}
}

// Find is a function to find a gorm result
func (gr *GormResult) Find(ctx context.Context, key string, dest interface{}, expiration time.Duration, f func() (interface{}, error)) error {
	err := r.GetGormResult(ctx, key, dest)
	if err == nil {
		return nil
	}

	res, err := f()
	if err != nil {
		return err
	}

	err = r.SetGormResult(ctx, key, res, expiration)
	if err != nil {
		return err
	}

	return gr.DB.Find(dest).Error
}

func ParseRedisOptions(redisURL string) *redis.Options {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil
	}
	return opt
}

var r *Redis
