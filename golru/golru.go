package golru

import (
	"context"
	"encoding/json"
	"time"

	"github.com/juliaqiuxy/wfcache"
	"github.com/manucorporat/golru"
)

type GoLRUStorage struct {
	golru *golru.Cache
	ttl   time.Duration
}

func Create(capacity int, ttl time.Duration) wfcache.StorageMaker {
	return CreateWithConfig(capacity, golru.DefaultLRUSamples, ttl)
}

func CreateWithConfig(capacity int, samples int, ttl time.Duration) wfcache.StorageMaker {
	return func() (wfcache.Storage, error) {
		golru := golru.New(capacity, golru.DefaultLRUSamples)

		s := &GoLRUStorage{
			golru: golru,
			ttl:   ttl,
		}

		return s, nil
	}
}

func (s *GoLRUStorage) TimeToLive() time.Duration {
	return s.ttl
}

func (s *GoLRUStorage) Get(ctx context.Context, key string) *wfcache.CacheItem {
	result := s.golru.Get(key)

	cacheItem := wfcache.CacheItem{}
	err := json.Unmarshal(result, &cacheItem)
	if err != nil {
		return nil
	}

	return &cacheItem
}

func (s *GoLRUStorage) BatchGet(ctx context.Context, keys []string) (results []*wfcache.CacheItem) {
	for _, key := range keys {
		m := s.Get(ctx, key)

		if m != nil {
			results = append(results, m)
		}
	}

	return results
}

func (s *GoLRUStorage) Set(ctx context.Context, key string, data []byte) error {
	item, err := json.Marshal(wfcache.CacheItem{
		Key:       key,
		Value:     data,
		ExpiresAt: time.Now().UTC().Add(s.ttl).Unix(),
	})

	if err != nil {
		return err
	}

	s.golru.Set(key, item)

	return nil
}

func (s *GoLRUStorage) BatchSet(ctx context.Context, pairs map[string][]byte) error {
	for key, data := range pairs {
		err := s.Set(ctx, key, data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *GoLRUStorage) Del(ctx context.Context, key string) error {
	s.golru.Del(key)

	return nil
}
