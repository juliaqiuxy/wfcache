package bigcache

// TODO(juliaqiuxy) Implement using https://github.com/allegro/bigcache

import (
	"context"
	"time"

	"github.com/juliaqiuxy/wfcache"
)

type BigCacheStorage struct{}

func Create(ttl time.Duration) wfcache.StorageMaker {
	return func() (wfcache.Storage, error) {
		s := &BigCacheStorage{}

		return s, nil
	}
}

func (s *BigCacheStorage) TimeToLive() time.Duration {
	return 0
}

func (s *BigCacheStorage) Get(ctx context.Context, key string) *wfcache.CacheItem {
	panic("bigcache: unimplemented")
}

func (s *BigCacheStorage) BatchGet(ctx context.Context, keys []string) (results []*wfcache.CacheItem) {
	panic("bigcache: unimplemented")
}

func (s *BigCacheStorage) Set(ctx context.Context, key string, data []byte) error {
	panic("bigcache: unimplemented")
}

func (s *BigCacheStorage) BatchSet(ctx context.Context, pairs map[string][]byte) error {
	panic("bigcache: unimplemented")
}

func (s *BigCacheStorage) Del(ctx context.Context, key string) error {
	panic("bigcache: unimplemented")
}
