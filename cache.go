package wfcache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/thoas/go-funk"
)

type CacheItem struct {
	Key       string `json:"key"`
	Value     []byte `json:"value"`
	ExpiresAt int64  `json:"expiresAt"`
}

type Storage interface {
	TimeToLive() time.Duration

	Get(ctx context.Context, key string) *CacheItem
	BatchGet(ctx context.Context, keys []string) []*CacheItem
	Set(ctx context.Context, key string, value []byte) error
	BatchSet(ctx context.Context, pairs map[string][]byte) error
	Del(ctx context.Context, key string) error
}

type StorageMaker func() (Storage, error)

type StartStorageOp func(ctx context.Context, opName string) interface{}
type FinishStorageOp func(interface{})
type Cache struct {
	storages []Storage

	makers []StorageMaker
	once   sync.Once

	startOperation  StartStorageOp
	finishOperation FinishStorageOp
}

var (
	ErrNotFulfilled       = errors.New("look up not fulfilled")
	ErrPartiallyFulfilled = errors.New("look up only partially fulfilled")
)

var nosop = func(ctx context.Context, opName string) interface{} {
	return nil
}
var nofop = func(input interface{}) {}

func Create(maker StorageMaker, otherMakers ...StorageMaker) (*Cache, error) {
	return CreateWithHooks(
		nosop,
		nofop,
		maker,
		otherMakers...)
}

func CreateWithHooks(sop StartStorageOp, fop FinishStorageOp, maker StorageMaker, otherMakers ...StorageMaker) (*Cache, error) {
	var c *Cache
	makers := append([]StorageMaker{maker}, otherMakers...)

	c = &Cache{
		startOperation:  sop,
		finishOperation: fop,
		makers:          makers,
	}

	return c, nil
}

func (c *Cache) ensureStorages() error {
	var err error

	c.once.Do(func() {
		c.storages = make([]Storage, len(c.makers))
		for i, makeStorage := range c.makers {
			storage, serr := makeStorage()
			if serr != nil {
				err = serr
				// TODO(juliaqiuxy) log error
				break
			}

			c.storages[i] = storage
		}
	})

	// Due to how once.Do works, if the first go routine to acquire
	// the lock takes a while to execute, other go routines trying to
	// ensure may temporarily get an empty storages slice to work with.
	// Since this isn't ideal, we ask to retry
	// TODO(juliaqiuxy) instead of an error, we could wait on once to
	// finish
	if len(c.storages) == 0 {
		return errors.New("cache not ready, try again soon")
	}

	return err
}

func (c *Cache) Get(key string) (*CacheItem, error) {
	return c.GetWithContext(context.Background(), key)
}

func (c *Cache) GetWithContext(ctx context.Context, key string) (*CacheItem, error) {
	err := c.ensureStorages()
	if err != nil {
		return nil, err
	}

	so := c.startOperation(ctx, "Get")
	defer c.finishOperation(so)

	missingKeyByStorage := map[Storage]string{}

	// start waterfall
	for _, storage := range c.storages {
		cacheItem := storage.Get(ctx, key)

		if cacheItem == nil {
			missingKeyByStorage[storage] = key
			continue
		} else {
			// prime previous storages
			for s := range missingKeyByStorage {
				s.Set(ctx, key, cacheItem.Value)
			}
		}

		// value := interface{}
		// err := json.Unmarshal(cacheItem.Value, value)

		// if err != nil {
		// 	return nil, err
		// }

		return cacheItem, nil
	}

	return nil, ErrNotFulfilled
}

func (c *Cache) BatchGet(keys []string) ([]*CacheItem, error) {
	return c.BatchGetWithContext(context.Background(), keys)
}

func (c *Cache) BatchGetWithContext(ctx context.Context, keys []string) ([]*CacheItem, error) {
	// TODO(juliaqiuxy) Detect dupes, empty keys, then bail

	err := c.ensureStorages()
	if err != nil {
		return nil, err
	}

	so := c.startOperation(ctx, "BatchGet")
	defer c.finishOperation(so)

	if len(keys) == 0 {
		return nil, errors.New("at least one key is required")
	}

	missingKeys := keys

	cacheItems := []*CacheItem{}

	missingKeysByStorage := map[Storage][]string{}

	// start waterfall
	for _, storage := range c.storages {
		mds := storage.BatchGet(ctx, missingKeys)

		if len(mds) != 0 {
			resolvedKeys := funk.Map(mds, func(md *CacheItem) string {
				return md.Key
			}).([]string)
			mKeys1, mKeys2 := funk.DifferenceString(resolvedKeys, missingKeys)
			missingKeys = append(mKeys1, mKeys2...)

			cacheItems = append(cacheItems, mds...)
		}

		if len(missingKeys) == 0 {
			break
		}

		missingKeysByStorage[storage] = missingKeys
	}

	// for _, cacheItem := range cacheItems {
	// 	if cacheItem != nil {
	// 		var m interface{}
	// 		json.Unmarshal(cacheItem.Value, &m)
	// 		*values = append(*values, m)
	// 	}
	// }

	if len(cacheItems) == 0 {
		return nil, ErrNotFulfilled
	}

	// prime previous storages
	for s, misses := range missingKeysByStorage {
		missedValues := map[string][]byte{}

		missedCacheItems := funk.Filter(cacheItems, func(md *CacheItem) bool {
			return funk.ContainsString(misses, md.Key)
		}).([]*CacheItem)

		for _, m := range missedCacheItems {
			missedValues[m.Key] = m.Value
		}

		if len(missedValues) != 0 {
			s.BatchSet(ctx, missedValues)
		}
	}

	if len(missingKeys) != 0 {
		return cacheItems, ErrPartiallyFulfilled
	}

	return cacheItems, nil
}

func (c *Cache) Set(key string, value interface{}) error {
	return c.SetWithContext(context.Background(), key, value)
}

func (c *Cache) SetWithContext(ctx context.Context, key string, value interface{}) error {
	err := c.ensureStorages()
	if err != nil {
		return err
	}

	so := c.startOperation(ctx, "Set")
	defer c.finishOperation(so)

	v, err := json.Marshal(value)
	if err != nil {
		return err
	}

	for _, storage := range c.storages {
		err := storage.Set(ctx, key, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cache) BatchSet(pairs map[string]interface{}) error {
	return c.BatchSetWithContext(context.Background(), pairs)
}

func (c *Cache) BatchSetWithContext(ctx context.Context, pairs map[string]interface{}) error {
	err := c.ensureStorages()
	if err != nil {
		return err
	}

	so := c.startOperation(ctx, "BatchSet")
	defer c.finishOperation(so)

	vPairs := map[string][]byte{}
	for key, value := range pairs {
		v, err := json.Marshal(value)
		if err != nil {
			return err
		}

		vPairs[key] = v
	}

	for _, storage := range c.storages {
		err := storage.BatchSet(ctx, vPairs)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cache) Del(key string) error {
	return c.DelWithContext(context.Background(), key)
}

func (c *Cache) DelWithContext(ctx context.Context, key string) error {
	err := c.ensureStorages()
	if err != nil {
		return err
	}

	so := c.startOperation(ctx, "Del")
	defer c.finishOperation(so)

	for _, storage := range c.storages {
		err := storage.Del(ctx, key)
		if err != nil {
			return err
		}
	}

	return nil
}
