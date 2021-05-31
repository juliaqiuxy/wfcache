# Waterfall Cache

[![GoDoc](https://godoc.org/github.com/juliaqiuxy/wfcache?status.svg)](https://godoc.org/github.com/juliaqiuxy/wfcache) [![npm](https://img.shields.io/github/license/juliaqiuxy/wfcache.svg?style=flat-square)](https://github.com/juliaqiuxy/wfcache/blob/master/LICENSE.md)

wfcache is a multi-layered cache with waterfall hit propagation and built-in storage adapters for DynamoDB, Redis, BigCache (in-memory)

> This project is under active development. Use at your own risk.

wfcache is effective for read-heavy workloads and it can be used both as a side-cache or a read-through/write-through cache. 

## Built-in Storage Adapters

| Package | Description | Eviction strategy
| --- | --- | --- |
| [Basic](basic/basic.go) | Basic in-memory storage | TTL (enforced on get) |
| [BigCache](https://github.com/allegro/bigcache) | BigCache | TTL/LRU |
| [Redis](https://github.com/go-redis/redis) | Redis | TTL/LRU |
| [DynamoDB](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb) | DynamoDB | [TTL](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/howitworks-ttl.html) |

## Installation

To retrieve wfcache, run:

```sh
$ go get github.com/juliaqiuxy/wfcache
```

To update to the latest version, run:

```sh
$ go get -u github.com/juliaqiuxy/wfcache
```

### Usage

```go
import "github.com/juliaqiuxy/wfcache"

wfcache.Create(
  onStartOperation,
  onFinishOperation,
  basicAdapter.Create(time.Minute),
  bigCacheAdapter.Create(time.Hour),
  dynamodbAdapter.Create(dynamodbClient, "my-cache-table", 3, 3, 24 * time.Hour),
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
  Get(ctx context.Context, key string) *Metadata
  BatchGet(ctx context.Context, keys []string) []*Metadata
  Set(ctx context.Context, key string, value []byte) error
  BatchSet(ctx context.Context, pairs map[string][]byte) error
  Del(ctx context.Context, key string) error
}
```