# Waterfall Cache

[![GoDoc](https://godoc.org/github.com/juliaqiuxy/wfcache?status.svg)](https://godoc.org/github.com/juliaqiuxy/wfcache) [![wfcache CI](https://github.com/juliaqiuxy/wfcache/actions/workflows/ci.yml/badge.svg)](https://github.com/juliaqiuxy/wfcache/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/juliaqiuxy/wfcache)](https://goreportcard.com/report/github.com/juliaqiuxy/wfcache) [![npm](https://img.shields.io/github/license/juliaqiuxy/wfcache.svg?style=flat-square)](https://github.com/juliaqiuxy/wfcache/blob/master/LICENSE.md)

wfcache is a multi-layered cache with waterfall hit propagation and built-in storage adapters for DynamoDB, Redis, BigCache (in-memory)

> This project is under active development. Use at your own risk.

wfcache is effective for read-heavy workloads and it can be used both as a side-cache or a read-through/write-through cache. 

## Built-in Storage Adapters

| Package | Description | Eviction strategy
| --- | --- | --- |
| [DynamoDB](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb) | DynamoDB | [TTL](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/howitworks-ttl.html) |
| [Redis](https://github.com/go-redis/redis) | Redis | TTL/[LRU](https://redis.io/topics/lru-cache) |
| [BigCache](https://github.com/allegro/bigcache) | Performant on heap storage with [minimal GC](https://github.com/allegro/bigcache#gc-pause-time) | TTL ([enforced on add](https://github.com/allegro/bigcache/issues/123#issuecomment-468902638)) |
| [GoLRU](https://github.com/manucorporat/golru) | In-memory storage with approximated LRU similar to Redis | TTL/LRU |
| [Basic](basic/basic.go) | Basic in-memory storage (not recommended) | TTL (enforced on get) |

## Installation

To retrieve wfcache, run:

```sh
$ go get github.com/juliaqiuxy/wfcache
```

### Usage

```go
import (
  "github.com/juliaqiuxy/wfcache"
  basic "github.com/juliaqiuxy/wfcache/basic"
  bigcache "github.com/juliaqiuxy/wfcache/bigcache"
  dynamodb "github.com/juliaqiuxy/wfcache/dynamodb"
)

c, err := wfcache.Create(
  basic.Create(5 * time.Minute),
  bigcache.Create(2 * time.Hour),
  dynamodb.Create(dynamodbClient, "my-cache-table", 24 * time.Hour),
)

items, err := c.BatchGet(keys)
if err == wfcache.ErrPartiallyFulfilled {
  fmt.Println("Somethings are missing")
}
```

## Usage with hooks

You can configure wfcache to notify you when each storage operation starts and finishes. This is useful when you want to do performance logging, tracing etc.

```go
import (
  "context"
  "github.com/juliaqiuxy/wfcache"
  "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
  basic "github.com/juliaqiuxy/wfcache/basic"
)

func onStartStorageOp(ctx context.Context, opName string) interface{} {
  span, _ := tracer.StartSpanFromContext(ctx, opName)
  return span
}

func onFinishStorageOp(span interface{}) {
  span.(ddtrace.Span).Finish()
}

wfcache.CreateWithHooks(
  onStartStorageOp,
  onFinishStorageOp,
  basic.Create(5 * time.Minute),
)
```

## How it works

The following steps outline how reads from wfcache work:

- When getting a value, wfcache tries to read it from the first storage layer (e.g. BigCache).
- If the storage layer is not populated with the requested key-value pair (cache miss), transparent to the application, wfcache notes the missing key and moves on to the next layer. This continues until all configured storage options are exhausted.
- When there is a cache hit, wfcache then primes each storage layer with a previously reported cache miss to make the data available for any subsequent reads.
- wfcache returns the key-value pair back to the application

If you want to use wfcache as read-through cache, you can implement a [custom adapter](#implementing-custom-adapters) for your source database and configure it as the last storage layer. In this setup, a cache miss only ever happens in intermediate storage layers (which are then primed as your source storage resolves values) but wfcache would always yield data.

When mutating wfcache, key-value pairs are written and removed from all storage layers. To mutate a specific storage layer in isolation, you can keep a refernece to it. However, this is not recommended as the interface is subject to change.

### Cache eviction

wfcache leaves it up to each storage layer to implement their eviction strategy. Built-in adapters use a combination of Time-to-Live (TTL) and Least Recently Used (LRU) algorithm to decide which items to evict. 

Also note that the built-in Basic storage is not meant for production use as the TTL enforcement only happens if and when a "stale" item is requested form the storage layer.

## Implementing Custom Adapters

For use cases where:

- you require a stroge adapter which is not [included](#built-in-storage-adapters) in wfcache, or
- you want to use wfcache as a read-through/write-through cache

it is trivial to extend wfcache by implementing the following adapter interface:

```go
type Storage interface {
  Get(ctx context.Context, key string) *CacheItem
  BatchGet(ctx context.Context, keys []string) []*CacheItem
  Set(ctx context.Context, key string, value []byte) error
  BatchSet(ctx context.Context, pairs map[string][]byte) error
  Del(ctx context.Context, key string) error
  BatchDel(ctx context.Context, keys []string) error
}
```
