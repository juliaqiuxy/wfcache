package redis_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/juliaqiuxy/wfcache"
	redisAdapter "github.com/juliaqiuxy/wfcache/redis"
)

var redisOnce sync.Once
var redisInstance *redis.Client

func RedisClient() *redis.Client {
	var redisDb = func() (*redis.Client, error) {
		db := redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
		})

		err := db.Ping(context.Background()).Err()
		if err != nil {
			return nil, err
		}

		return db, nil
	}

	redisOnce.Do(func() {
		var err error
		redisInstance, err = redisDb()

		if err != nil {
			panic(err)
		}
	})

	return redisInstance
}

func TestRedis(t *testing.T) {
	r := RedisClient()

	c, _ := wfcache.Create(
		redisAdapter.Create(r, 6*time.Hour),
	)

	key := "my_key"
	val := "my_value"

	c.Set(key, val)

	items, err := c.BatchGet([]string{key})

	if len(items) != 1 {
		t.Errorf("Received %v items, expected 1", len(items))
	}

	var str string
	json.Unmarshal(items[0].Value, &str)

	if str != val {
		t.Errorf("Received %v (type %v), expected %v (type %v)", str, reflect.TypeOf(str), val, reflect.TypeOf(val))
	}

	fmt.Println(items, str, err)
}
