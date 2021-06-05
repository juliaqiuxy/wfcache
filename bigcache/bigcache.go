package bigcache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/juliaqiuxy/wfcache"
)

type BigCacheStorage struct {
	bigCache *bigcache.BigCache
	ttl      time.Duration
}

func Create(ttl time.Duration) wfcache.StorageMaker {
	return CreateWithConfig(bigcache.DefaultConfig(ttl))
}

func CreateWithConfig(conf bigcache.Config) wfcache.StorageMaker {
	return func() (wfcache.Storage, error) {
		bigCache, err := bigcache.NewBigCache(conf)

		if err != nil {
			return nil, err
		}

		s := &BigCacheStorage{
			bigCache: bigCache,
			ttl:      conf.LifeWindow,
		}

		return s, nil
	}
}

func (s *BigCacheStorage) TimeToLive() time.Duration {
	return s.ttl
}

func (s *BigCacheStorage) Get(ctx context.Context, key string) *wfcache.CacheItem {
	result, err := s.bigCache.Get(key)
	if err != nil {
		return nil
	}

	cacheItem := wfcache.CacheItem{}
	err = json.Unmarshal(result, &cacheItem)
	if err != nil {
		return nil
	}

	return &cacheItem
}

func (s *BigCacheStorage) BatchGet(ctx context.Context, keys []string) (results []*wfcache.CacheItem) {
	for _, key := range keys {
		m := s.Get(ctx, key)

		if m != nil {
			results = append(results, m)
		}
	}

	return results
}

func (s *BigCacheStorage) Set(ctx context.Context, key string, data []byte) error {
	item, err := json.Marshal(wfcache.CacheItem{
		Key:       key,
		Value:     data,
		ExpiresAt: time.Now().UTC().Add(s.ttl).Unix(),
	})

	if err != nil {
		return err
	}

	err = s.bigCache.Set(key, item)
	if err != nil {
		return err
	}

	return nil
}

func (s *BigCacheStorage) BatchSet(ctx context.Context, pairs map[string][]byte) error {
	for key, data := range pairs {
		err := s.Set(ctx, key, data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *BigCacheStorage) Del(ctx context.Context, key string) error {
	err := s.bigCache.Delete(key)

	if err != nil {
		return err
	}

	return nil
}
