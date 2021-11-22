package wfcache_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/go-redis/redis/v8"
	"github.com/juliaqiuxy/wfcache"
	basicAdapter "github.com/juliaqiuxy/wfcache/basic"
	bigCacheAdapter "github.com/juliaqiuxy/wfcache/bigcache"
	dynamodbAdapter "github.com/juliaqiuxy/wfcache/dynamodb"
	goLruAdapter "github.com/juliaqiuxy/wfcache/golru"
	redisAdapter "github.com/juliaqiuxy/wfcache/redis"
)

var dynamodbOnce sync.Once
var dynamodbInstance *dynamodb.DynamoDB

var redisOnce sync.Once
var redisInstance *redis.Client

func TestWfCacheSetGetWithBasicAdapter(t *testing.T) {
	c, _ := wfcache.New(
		basicAdapter.Create(5 * time.Minute),
	)

	key := "my_key"
	val := "my_value"

	c.Set(key, val)

	item, err := c.Get(key)

	var str string
	json.Unmarshal(item.Value, &str)

	if str != val {
		t.Errorf("Received %v (type %v), expected %v (type %v)", str, reflect.TypeOf(str), val, reflect.TypeOf(val))
	}

	storages, _ := c.Storages()

	fmt.Println(item, str, err, storages, len(storages))
}

func TestWfCacheSetBatchGetWithBasicAdapter(t *testing.T) {
	c, _ := wfcache.New(
		basicAdapter.Create(5 * time.Minute),
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

	storages, _ := c.Storages()

	fmt.Println(items, str, err, storages, len(storages))
}

func TestWfCacheBatchSetBatchGetWithBasicAdapter(t *testing.T) {
	c, _ := wfcache.New(
		basicAdapter.Create(5 * time.Minute),
	)

	keys := []string{
		"my_key1",
		"my_key2",
	}

	pairs := map[string]interface{}{
		"my_key1": "my_value1",
		"my_key2": "my_value2",
	}

	c.BatchSet(pairs)

	items, err := c.BatchGet(keys)

	if len(items) != 2 {
		t.Errorf("Received %v items, expected 2", len(items))
	}

	var returnedPairs []map[string]interface{}
	for _, item := range items {
		var value string
		json.Unmarshal(item.Value, &value)
		key := item.Key

		returnedPairs = append(returnedPairs, map[string]interface{}{
			key: value,
		})
	}

	for _, returnedPair := range returnedPairs {
		var key string

		for pairKey := range returnedPair {
			for _, k := range keys {
				if k == pairKey {
					key = k
				}
			}
		}

		if returnedPair[key] != pairs[key] {
			t.Errorf("Received %v (type %v), expected %v (type %v)", returnedPair[key], reflect.TypeOf(returnedPair[key]), pairs[key], reflect.TypeOf(pairs[key]))
		}
	}

	storages, _ := c.Storages()

	fmt.Println(items, pairs, err, storages, len(storages))
}

func TestWfCacheSetGetWithBigCacheAdapter(t *testing.T) {
	c, _ := wfcache.New(
		bigCacheAdapter.Create(30 * time.Minute),
	)

	key := "my_key"
	val := "my_value"

	c.Set(key, val)

	item, err := c.Get(key)

	var str string
	json.Unmarshal(item.Value, &str)

	if str != val {
		t.Errorf("Received %v (type %v), expected %v (type %v)", str, reflect.TypeOf(str), val, reflect.TypeOf(val))
	}

	storages, _ := c.Storages()

	fmt.Println(item, str, err, storages, len(storages))
}

func TestWfCacheSetBatchGetWithBigCacheAdapter(t *testing.T) {
	c, _ := wfcache.New(
		bigCacheAdapter.Create(30 * time.Minute),
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

	storages, _ := c.Storages()

	fmt.Println(items, str, err, storages, len(storages))
}

func TestWfCacheBatchSetBatchGetWithBigCacheAdapter(t *testing.T) {
	c, _ := wfcache.New(
		bigCacheAdapter.Create(30 * time.Minute),
	)

	keys := []string{
		"my_key1",
		"my_key2",
	}

	pairs := map[string]interface{}{
		"my_key1": "my_value1",
		"my_key2": "my_value2",
	}

	c.BatchSet(pairs)

	items, err := c.BatchGet(keys)

	if len(items) != 2 {
		t.Errorf("Received %v items, expected 2", len(items))
	}

	var returnedPairs []map[string]interface{}
	for _, item := range items {
		var value string
		json.Unmarshal(item.Value, &value)
		key := item.Key

		returnedPairs = append(returnedPairs, map[string]interface{}{
			key: value,
		})
	}

	for _, returnedPair := range returnedPairs {
		var key string

		for pairKey := range returnedPair {
			for _, k := range keys {
				if k == pairKey {
					key = k
				}
			}
		}

		if returnedPair[key] != pairs[key] {
			t.Errorf("Received %v (type %v), expected %v (type %v)", returnedPair[key], reflect.TypeOf(returnedPair[key]), pairs[key], reflect.TypeOf(pairs[key]))
		}
	}

	storages, _ := c.Storages()

	fmt.Println(items, pairs, err, storages, len(storages))
}

func TestWfCacheSetGetWithGoLruAdapter(t *testing.T) {
	c, _ := wfcache.New(
		goLruAdapter.Create(64, 30*time.Minute),
	)

	key := "my_key"
	val := "my_value"

	c.Set(key, val)

	item, err := c.Get(key)

	var str string
	json.Unmarshal(item.Value, &str)

	if str != val {
		t.Errorf("Received %v (type %v), expected %v (type %v)", str, reflect.TypeOf(str), val, reflect.TypeOf(val))
	}

	storages, _ := c.Storages()

	fmt.Println(item, str, err, storages, len(storages))
}

func TestWfCacheSetBatchGetWithGoLruAdapter(t *testing.T) {
	c, _ := wfcache.New(
		goLruAdapter.Create(64, 30*time.Minute),
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

	storages, _ := c.Storages()

	fmt.Println(items, str, err, storages, len(storages))
}

func TestWfCacheBatchSetBatchGetWithGoLruAdapter(t *testing.T) {
	c, _ := wfcache.New(
		goLruAdapter.Create(64, 30*time.Minute),
	)

	keys := []string{
		"my_key1",
		"my_key2",
	}

	pairs := map[string]interface{}{
		"my_key1": "my_value1",
		"my_key2": "my_value2",
	}

	c.BatchSet(pairs)

	items, err := c.BatchGet(keys)

	if len(items) != 2 {
		t.Errorf("Received %v items, expected 2", len(items))
	}

	var returnedPairs []map[string]interface{}
	for _, item := range items {
		var value string
		json.Unmarshal(item.Value, &value)
		key := item.Key

		returnedPairs = append(returnedPairs, map[string]interface{}{
			key: value,
		})
	}

	for _, returnedPair := range returnedPairs {
		var key string

		for pairKey := range returnedPair {
			for _, k := range keys {
				if k == pairKey {
					key = k
				}
			}
		}

		if returnedPair[key] != pairs[key] {
			t.Errorf("Received %v (type %v), expected %v (type %v)", returnedPair[key], reflect.TypeOf(returnedPair[key]), pairs[key], reflect.TypeOf(pairs[key]))
		}
	}

	storages, _ := c.Storages()

	fmt.Println(items, pairs, err, storages, len(storages))
}

func DynamodbClient() *dynamodb.DynamoDB {
	var newLocalDynamodb = func() *dynamodb.DynamoDB {
		dynamodbHost, ok := os.LookupEnv("DYNAMODB_HOST")
		if !ok {
			dynamodbHost = "http://localhost:8000"
		}

		s, err := session.NewSessionWithOptions(session.Options{
			Profile: "dynamodb-local",
			Config: aws.Config{
				Region:      aws.String("dev-region"),
				Endpoint:    aws.String(dynamodbHost),
				Credentials: credentials.NewStaticCredentials("dev-key-id", "dev-secret-key", ""),
			},
		})

		if err != nil {
			log.Fatal(err)
			return nil
		}

		return dynamodb.New(s)
	}

	dynamodbOnce.Do(func() {
		dynamodbInstance = newLocalDynamodb()
	})

	return dynamodbInstance
}

var dynamodbClient = DynamodbClient()

func TestWfCacheSetGetWithDynamoDbAdapter(t *testing.T) {
	c, _ := wfcache.New(
		dynamodbAdapter.Create(dynamodbClient, "tests", 6*time.Hour),
	)

	key := "my_key"
	val := "my_value"

	c.Set(key, val)

	item, err := c.Get(key)

	var str string
	json.Unmarshal(item.Value, &str)

	if str != val {
		t.Errorf("Received %v (type %v), expected %v (type %v)", str, reflect.TypeOf(str), val, reflect.TypeOf(val))
	}

	storages, _ := c.Storages()

	fmt.Println(item, str, err, storages, len(storages))
}

func TestWfCacheSetBatchGetWithDynamoDbAdapter(t *testing.T) {
	c, _ := wfcache.New(
		dynamodbAdapter.Create(dynamodbClient, "tests", 6*time.Hour),
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

	storages, _ := c.Storages()

	fmt.Println(items, str, err, storages, len(storages))
}

func TestWfCacheBatchSetBatchGetWithDynamoDbAdapter(t *testing.T) {
	c, _ := wfcache.New(
		dynamodbAdapter.Create(dynamodbClient, "tests", 6*time.Hour),
	)

	keys := []string{
		"my_key1",
		"my_key2",
	}

	pairs := map[string]interface{}{
		"my_key1": "my_value1",
		"my_key2": "my_value2",
	}

	c.BatchSet(pairs)

	items, err := c.BatchGet(keys)

	if len(items) != 2 {
		t.Errorf("Received %v items, expected 2", len(items))
	}

	var returnedPairs []map[string]interface{}
	for _, item := range items {
		var value string
		json.Unmarshal(item.Value, &value)
		key := item.Key

		returnedPairs = append(returnedPairs, map[string]interface{}{
			key: value,
		})
	}

	for _, returnedPair := range returnedPairs {
		var key string

		for pairKey := range returnedPair {
			for _, k := range keys {
				if k == pairKey {
					key = k
				}
			}
		}

		if returnedPair[key] != pairs[key] {
			t.Errorf("Received %v (type %v), expected %v (type %v)", returnedPair[key], reflect.TypeOf(returnedPair[key]), pairs[key], reflect.TypeOf(pairs[key]))
		}
	}

	storages, _ := c.Storages()

	fmt.Println(items, pairs, err, storages, len(storages))
}

func RedisClient() *redis.Client {
	var redisDb = func() (*redis.Client, error) {
		redisHost, ok := os.LookupEnv("REDIS_HOST")
		if !ok {
			redisHost = "localhost:6379"
		}

		db := redis.NewClient(&redis.Options{
			Addr:     redisHost,
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

var r = RedisClient()

func TestWfCacheSetGetWithRedisAdapter(t *testing.T) {
	c, _ := wfcache.New(
		redisAdapter.Create(r, 6*time.Hour),
	)

	key := "my_key"
	val := "my_value"

	c.Set(key, val)

	item, err := c.Get(key)

	var str string
	json.Unmarshal(item.Value, &str)

	if str != val {
		t.Errorf("Received %v (type %v), expected %v (type %v)", str, reflect.TypeOf(str), val, reflect.TypeOf(val))
	}

	storages, _ := c.Storages()

	fmt.Println(item, str, err, storages, len(storages))
}

func TestWfCacheSetBatchGetWithRedisAdapter(t *testing.T) {
	c, _ := wfcache.New(
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

	storages, _ := c.Storages()

	fmt.Println(items, str, err, storages, len(storages))
}

func TestWfCacheBatchSetBatchGetWithRedisAdapter(t *testing.T) {
	c, _ := wfcache.New(
		redisAdapter.Create(r, 6*time.Hour),
	)

	keys := []string{
		"my_key1",
		"my_key2",
	}

	pairs := map[string]interface{}{
		"my_key1": "my_value1",
		"my_key2": "my_value2",
	}

	c.BatchSet(pairs)

	items, err := c.BatchGet(keys)

	if len(items) != 2 {
		t.Errorf("Received %v items, expected 2", len(items))
	}

	var returnedPairs []map[string]interface{}
	for _, item := range items {
		var value string
		json.Unmarshal(item.Value, &value)
		key := item.Key

		returnedPairs = append(returnedPairs, map[string]interface{}{
			key: value,
		})
	}

	for _, returnedPair := range returnedPairs {
		var key string

		for pairKey := range returnedPair {
			for _, k := range keys {
				if k == pairKey {
					key = k
				}
			}
		}

		if returnedPair[key] != pairs[key] {
			t.Errorf("Received %v (type %v), expected %v (type %v)", returnedPair[key], reflect.TypeOf(returnedPair[key]), pairs[key], reflect.TypeOf(pairs[key]))
		}
	}

	storages, _ := c.Storages()

	fmt.Println(items, pairs, err, storages, len(storages))
}

func TestWfCacheSetGetWithAllAdapters(t *testing.T) {
	c, _ := wfcache.New(
		goLruAdapter.Create(64, 30*time.Minute),
		bigCacheAdapter.Create(30*time.Minute),
		basicAdapter.Create(5*time.Minute),
		dynamodbAdapter.Create(dynamodbClient, "tests", 6*time.Hour),
		redisAdapter.Create(r, 6*time.Hour),
	)

	key := "my_key"
	val := "my_value"

	c.Set(key, val)

	item, err := c.Get(key)

	var str string
	json.Unmarshal(item.Value, &str)

	if str != val {
		t.Errorf("Received %v (type %v), expected %v (type %v)", str, reflect.TypeOf(str), val, reflect.TypeOf(val))
	}

	storages, _ := c.Storages()

	fmt.Println(item, str, err, storages, len(storages))
}

func TestWfCacheSetBatchGetWithAllAdapters(t *testing.T) {
	c, _ := wfcache.New(
		goLruAdapter.Create(64, 30*time.Minute),
		bigCacheAdapter.Create(30*time.Minute),
		basicAdapter.Create(5*time.Minute),
		dynamodbAdapter.Create(dynamodbClient, "tests", 6*time.Hour),
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

	storages, _ := c.Storages()

	fmt.Println(items, str, err, storages, len(storages))
}

func TestWfCacheBatchSetBatchGetWithAllAdapters(t *testing.T) {
	c, _ := wfcache.New(
		goLruAdapter.Create(64, 30*time.Minute),
		bigCacheAdapter.Create(30*time.Minute),
		basicAdapter.Create(5*time.Minute),
		dynamodbAdapter.Create(dynamodbClient, "tests", 6*time.Hour),
		redisAdapter.Create(r, 6*time.Hour),
	)

	keys := []string{
		"my_key1",
		"my_key2",
	}

	pairs := map[string]interface{}{
		"my_key1": "my_value1",
		"my_key2": "my_value2",
	}

	c.BatchSet(pairs)

	items, err := c.BatchGet(keys)

	if len(items) != 2 {
		t.Errorf("Received %v items, expected 2", len(items))
	}

	var returnedPairs []map[string]interface{}
	for _, item := range items {
		var value string
		json.Unmarshal(item.Value, &value)
		key := item.Key

		returnedPairs = append(returnedPairs, map[string]interface{}{
			key: value,
		})
	}

	for _, returnedPair := range returnedPairs {
		var key string

		for pairKey := range returnedPair {
			for _, k := range keys {
				if k == pairKey {
					key = k
				}
			}
		}

		if returnedPair[key] != pairs[key] {
			t.Errorf("Received %v (type %v), expected %v (type %v)", returnedPair[key], reflect.TypeOf(returnedPair[key]), pairs[key], reflect.TypeOf(pairs[key]))
		}
	}

	storages, _ := c.Storages()

	fmt.Println(items, pairs, err, storages, len(storages))
}
