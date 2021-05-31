package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/juliaqiuxy/wfcache"
)

const NoTTL time.Duration = -1

type BasicStorage struct {
	pairs map[string]*wfcache.CacheItem
	ttl   time.Duration

	mutex sync.RWMutex
}

func Create(ttl time.Duration) wfcache.StorageMaker {
	return func() (wfcache.Storage, error) {
		if ttl == 0 {
			return nil, errors.New("basic: storage requires a ttl")
		}

		s := &BasicStorage{
			pairs: make(map[string]*wfcache.CacheItem),
			ttl:   ttl,
		}

		return s, nil
	}
}

func (s *BasicStorage) TimeToLive() time.Duration {
	return s.ttl
}

func (s *BasicStorage) Get(ctx context.Context, key string) *wfcache.CacheItem {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	m, found := s.pairs[key]

	if found {
		if s.ttl == NoTTL || time.Now().UTC().Before(time.Unix(m.ExpiresAt, 0)) {
			return m
		} else {
			s.Del(ctx, key)
		}
	}

	return nil
}

func (s *BasicStorage) BatchGet(ctx context.Context, keys []string) (results []*wfcache.CacheItem) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, key := range keys {
		m := s.Get(ctx, key)

		if m != nil {
			results = append(results, m)
		}
	}

	return results
}

func (s *BasicStorage) Set(ctx context.Context, key string, data []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.pairs[key] = &wfcache.CacheItem{
		Key:       key,
		Value:     data,
		ExpiresAt: time.Now().UTC().Add(s.ttl).Unix(),
	}

	return nil
}

func (s *BasicStorage) BatchSet(ctx context.Context, pairs map[string][]byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for key, data := range pairs {
		s.pairs[key] = &wfcache.CacheItem{
			Key:       key,
			Value:     data,
			ExpiresAt: time.Now().UTC().Add(s.ttl).Unix(),
		}
	}

	return nil
}

func (s *BasicStorage) Del(ctx context.Context, key string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.pairs, key)

	return nil
}
