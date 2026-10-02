package repository

import (
	"context"
	"errors"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/redis/go-redis/v9"
)

const autoResponseAffinityPrefix = "openai:auto_response_affinity:"

var _ service.AutoResponseAffinityCache = (*gatewayCache)(nil)

func (c *gatewayCache) StoreAutoResponseAffinity(ctx context.Context, index string, data []byte, ttl time.Duration) error {
	stored, err := c.rdb.SetNX(ctx, autoResponseAffinityPrefix+index, data, ttl).Result()
	if err != nil {
		return err
	}
	if !stored {
		return errors.New("response affinity already exists")
	}
	return nil
}

func (c *gatewayCache) LoadAutoResponseAffinity(ctx context.Context, index string) ([]byte, error) {
	data, err := c.rdb.Get(ctx, autoResponseAffinityPrefix+index).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	return data, err
}
