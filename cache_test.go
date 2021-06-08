package wfcache_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/juliaqiuxy/wfcache"
	basicAdapter "github.com/juliaqiuxy/wfcache/basic"
	bigCacheAdapter "github.com/juliaqiuxy/wfcache/bigcache"
)

func TestWfCache(t *testing.T) {
	c, _ := wfcache.Create(
		basicAdapter.Create(5*time.Minute),
		bigCacheAdapter.Create(30*time.Minute),
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
