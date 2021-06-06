package redis

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-redis/redis/v8"
	"github.com/juliaqiuxy/wfcache"
	"github.com/thoas/go-funk"
)

type RedisStorage struct {
	redisClient *redis.Client
	ttl         time.Duration
}

const maxReadOps = 200
const maxWriteOps = 200

func Create(redisClient *redis.Client, ttl time.Duration) wfcache.StorageMaker {
	return func() (wfcache.Storage, error) {
		s := &RedisStorage{
			redisClient: redisClient,
			ttl:         ttl,
		}

		return s, nil
	}
}

func (s *RedisStorage) TimeToLive() time.Duration {
	return s.ttl
}

func (s *RedisStorage) Get(ctx context.Context, key string) *wfcache.CacheItem {
	result, err := s.redisClient.Get(ctx, key).Bytes()

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

func (s *RedisStorage) BatchGet(ctx context.Context, keys []string) (results []*wfcache.CacheItem) {
	queue := keys

process:
	maxItems := int(math.Min(maxReadOps, float64(len(queue))))
	next := queue[0:maxItems]
	queue = queue[maxItems:]

	var items []interface{}
	err := withRetry(ctx, func() error {
		var err error

		items, err = s.redisClient.MGet(ctx, next...).Result()

		return err
	})

	if err != nil {
		// TODO(juliaqiuxy) log debug
		return nil
	}

	for _, item := range items {
		if item != nil {
			cacheItem := wfcache.CacheItem{}
			err = json.Unmarshal([]byte(item.(string)), &cacheItem)

			if err != nil {
				// TODO(juliaqiuxy) log debug
				return nil
			}

			results = append(results, &cacheItem)
		}
	}

	if len(queue) != 0 {
		goto process
	}

	return results
}

func (s *RedisStorage) Set(ctx context.Context, key string, data []byte) error {
	item, err := json.Marshal(wfcache.CacheItem{
		Key:       key,
		Value:     data,
		ExpiresAt: time.Now().UTC().Add(s.ttl).Unix(),
	})

	if err != nil {
		return err
	}

	err = s.redisClient.Set(ctx, key, item, s.ttl).Err()
	if err != nil {
		return err
	}

	return nil
}

func (s *RedisStorage) BatchSet(ctx context.Context, pairs map[string][]byte) error {
	queue := funk.Keys(pairs).([]string)

process:
	maxItems := int(math.Min(maxWriteOps, float64(len(queue))))
	next := queue[0:maxItems]
	queue = queue[maxItems:]

	nextPairs := funk.Reduce(next, func(acc map[string]interface{}, key string) map[string]interface{} {
		item, _ := json.Marshal(wfcache.CacheItem{
			Key:       key,
			Value:     pairs[key],
			ExpiresAt: time.Now().UTC().Add(s.ttl).Unix(),
		})
		acc[key] = item

		return acc
	}, map[string]interface{}{})

	err := withRetry(ctx, func() error {
		pipe := s.redisClient.TxPipeline()

		// MSet doesn't support TTL. So try to do it all in a round-trip
		pipe.MSet(ctx, nextPairs.(map[string]interface{}))

		for _, key := range next {
			pipe.Expire(ctx, key, s.ttl)
		}

		_, err := pipe.Exec(ctx)

		return err
	})

	if err != nil {
		return err
	}

	if len(queue) != 0 {
		goto process
	}

	return nil
}

func (s *RedisStorage) Del(ctx context.Context, key string) error {
	return s.BatchDel(ctx, []string{key})
}

func (s *RedisStorage) BatchDel(ctx context.Context, keys []string) error {
	err := s.redisClient.Del(ctx, keys...).Err()

	if err != nil {
		return err
	}

	return nil
}

func withRetry(ctx context.Context, fn func() error) (err error) {
	var wait time.Duration

	b := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)

	for {
		err = fn()

		if err == nil {
			return nil
		}

		if !isRetriable(err) {
			return err
		}

		wait = b.NextBackOff()

		if wait == backoff.Stop {
			return err
		}

		err = sleepWithContext(ctx, wait)
		if err != nil {
			return err
		}
	}
}

func isRetriable(err error) bool {
	// https://github.com/go-redis/redis/blob/v8.10.0/error.go
	return false
}

func sleepWithContext(ctx context.Context, dur time.Duration) error {
	t := time.NewTimer(dur)
	defer t.Stop()

	select {
	case <-t.C:
		break
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
