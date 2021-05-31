package redis

import (
	"context"
	"time"

	"github.com/juliaqiuxy/wfcache"
)

type RedisStorage struct{}

func Create(ttl time.Duration) wfcache.StorageMaker {
	return func() (wfcache.Storage, error) {
		s := &RedisStorage{}

		return s, nil
	}
}

func (s *RedisStorage) TimeToLive() time.Duration {
	return 0
}

func (s *RedisStorage) Get(ctx context.Context, key string) *wfcache.CacheItem {
	panic("redis: unimplemented")
}

func (s *RedisStorage) BatchGet(ctx context.Context, keys []string) (results []*wfcache.CacheItem) {
	panic("redis: unimplemented")
}

func (s *RedisStorage) Set(ctx context.Context, key string, data []byte) error {
	panic("redis: unimplemented")
}

func (s *RedisStorage) BatchSet(ctx context.Context, pairs map[string][]byte) error {
	panic("redis: unimplemented")
}

func (s *RedisStorage) Del(ctx context.Context, key string) error {
	panic("redis: unimplemented")
}
