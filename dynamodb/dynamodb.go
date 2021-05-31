package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/cenkalti/backoff/v4"
	"github.com/juliaqiuxy/wfcache"
	"github.com/thoas/go-funk"
)

type DynamoDbStorage struct {
	dynamodbClient dynamodbiface.DynamoDBAPI
	tableName      string
	ttl            time.Duration
}

const maxReadOps = 100
const maxWriteOps = 25

func prepareDynamoDbTableIfNotExists(dynamodbClient dynamodbiface.DynamoDBAPI, tableName string, readCapacityUnits int64, writeCapacityUnits int64) error {
	_, err := dynamodbClient.CreateTable(&dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("key"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("key"),
				KeyType:       aws.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(readCapacityUnits),
			WriteCapacityUnits: aws.Int64(writeCapacityUnits),
		},
		TableName: aws.String(tableName),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != dynamodb.ErrCodeResourceInUseException {
				return err
			}
		}
	} else {
		// TODO(juliaqiuxy) perform describe on the table to ensure configuration passed
		// by other procs are consistent, or panic with a good explanation as to why e.g.
		// Underlying dynamodb table can't be shared across different stroage configuration
		dynamodbClient.UpdateTimeToLive(&dynamodb.UpdateTimeToLiveInput{
			TableName: aws.String(tableName),
			TimeToLiveSpecification: &dynamodb.TimeToLiveSpecification{
				Enabled:       aws.Bool(true),
				AttributeName: aws.String("expiresAt"),
			},
		})
	}

	return nil
}

func Create(dynamodbClient dynamodbiface.DynamoDBAPI, tableName string, readCapacityUnits int64, writeCapacityUnits int64, ttl time.Duration) wfcache.StorageMaker {
	return func() (wfcache.Storage, error) {
		if dynamodbClient == nil {
			return nil, errors.New("dynamodb requires a client")
		}
		if tableName == "" {
			return nil, errors.New("dynamodb storage requires a table name")
		}
		if ttl == 0 {
			return nil, errors.New("dynamodb storage requires a ttl")
		}

		s := &DynamoDbStorage{
			dynamodbClient: dynamodbClient,
			tableName:      tableName,
			ttl:            ttl,
		}

		_, err := dynamodbClient.DescribeTable(&dynamodb.DescribeTableInput{
			TableName: aws.String(s.tableName),
		})

		if err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				switch awserr.Code() {
				case dynamodb.ErrCodeResourceNotFoundException:
					err = prepareDynamoDbTableIfNotExists(
						dynamodbClient,
						tableName,
						readCapacityUnits,
						writeCapacityUnits,
					)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		return s, nil
	}
}

func (s *DynamoDbStorage) TimeToLive() time.Duration {
	return s.ttl
}

func (s *DynamoDbStorage) Get(ctx context.Context, key string) *wfcache.CacheItem {
	result, err := s.dynamodbClient.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String(key),
			},
		},
	})

	if err != nil || result.Item == nil {
		return nil
	}

	cacheItem := wfcache.CacheItem{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &cacheItem)

	if err != nil {
		return nil
	}

	return &cacheItem
}

// If you request more than 100 items, BatchGetItem returns a ValidationException
// with the message "Too many items requested for the BatchGetItem call."
func (s *DynamoDbStorage) BatchGet(ctx context.Context, keys []string) (results []*wfcache.CacheItem) {
	queue := keys

process:
	maxItems := int(math.Min(maxReadOps, float64(len(queue))))
	next := queue[0:maxItems]
	queue = queue[maxItems:]

	mapOfAttrKeys := []map[string]*dynamodb.AttributeValue{}
	for _, key := range next {
		mapOfAttrKeys = append(mapOfAttrKeys, map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String(key),
			},
		})
	}

	var result *dynamodb.BatchGetItemOutput
	err := withRetry(ctx, func() error {
		var err error

		// TODO use BatchWriteItemWithContext
		result, err = s.dynamodbClient.BatchGetItem(&dynamodb.BatchGetItemInput{
			RequestItems: map[string]*dynamodb.KeysAndAttributes{
				s.tableName: {
					Keys: mapOfAttrKeys,
				},
			},
		})

		return err
	})

	if err != nil {
		// TODO(juliaqiuxy) log debug
		return nil
	}

	for _, table := range result.Responses {
		for _, item := range table {
			cacheItem := wfcache.CacheItem{}
			err = dynamodbattribute.UnmarshalMap(item, &cacheItem)

			if err != nil {
				// TODO(juliaqiuxy) log debug
				return nil
			}

			results = append(results, &cacheItem)
		}
	}

	// if the results exceeds 16MB, put the unprocessed keys back on in the queue
	unprocessedTable := result.UnprocessedKeys[s.tableName]

	if unprocessedTable != nil && unprocessedTable.Keys != nil {
		unprocessedKeys := funk.Map(unprocessedTable.Keys, func(item map[string]*dynamodb.AttributeValue) string {
			return *item["key"].S
		}).([]string)

		queue = append(queue, unprocessedKeys...)
	}

	if len(queue) != 0 {
		goto process
	}

	return results
}

func (s *DynamoDbStorage) Set(ctx context.Context, key string, data []byte) error {
	item, err := dynamodbattribute.MarshalMap(wfcache.CacheItem{
		Key:       key,
		Value:     data,
		ExpiresAt: time.Now().UTC().Add(s.ttl).Unix(),
	})

	if err != nil {
		return err
	}

	_, err = s.dynamodbClient.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *DynamoDbStorage) BatchSet(ctx context.Context, pairs map[string][]byte) error {

	queue := funk.Keys(pairs).([]string)

process:
	maxItems := int(math.Min(maxWriteOps, float64(len(queue))))
	next := queue[0:maxItems]
	queue = queue[maxItems:]

	mapOfAttrKeys := []*dynamodb.WriteRequest{}
	for _, key := range next {
		item, err := dynamodbattribute.MarshalMap(wfcache.CacheItem{
			Key:       key,
			Value:     pairs[key],
			ExpiresAt: time.Now().UTC().Add(s.ttl).Unix(),
		})

		if err != nil {
			return err
		}

		mapOfAttrKeys = append(
			mapOfAttrKeys,
			&dynamodb.WriteRequest{
				PutRequest: &dynamodb.PutRequest{
					Item: item,
				},
			},
		)
	}

	var result *dynamodb.BatchWriteItemOutput
	err := withRetry(ctx, func() error {
		var err error

		// TODO use BatchWriteItemWithContext
		result, err = s.dynamodbClient.BatchWriteItem(&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				s.tableName: mapOfAttrKeys,
			},
		})
		return err
	})

	if err != nil {
		return fmt.Errorf(errDynamodbBatchWrite, err)
	}

	// if we have unprocessed items due to dynamodb limits,
	// put them back in the queue
	unprocessedItems := result.UnprocessedItems[s.tableName]

	if unprocessedItems != nil {
		unprocessedKeys := funk.Map(unprocessedItems, func(item *dynamodb.WriteRequest) string {
			return *item.PutRequest.Item["key"].S
		}).([]string)

		queue = append(queue, unprocessedKeys...)
	}

	if len(queue) != 0 {
		goto process
	}

	return nil
}

func (s *DynamoDbStorage) Del(ctx context.Context, key string) error {
	_, err := s.dynamodbClient.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String(key),
			},
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func withRetry(ctx aws.Context, fn func() error) (err error) {
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

		err = aws.SleepWithContext(ctx, wait)
		if err != nil {
			return err
		}
	}
}

func isRetriable(err error) bool {
	if aerr, ok := err.(awserr.RequestFailure); ok {
		switch aerr.StatusCode() {
		case
			429, // error caused due to too many requests
			500, // DynamoDB could not process, retry
			502, // Bad Gateway error should be throttled
			503, // Caused when service is unavailable
			504: // Error occurred due to gateway timeout
			return true
		}
	}

	if request.IsErrorThrottle(err) || request.IsErrorRetryable(err) {
		return true
	}

	return false
}

const errDynamodbBatchWrite = `error: %s

DynamoDB rejects the entire batch write operation when one or more of the following is true:

* There are more than 25 requests in the batch.

* Any individual item in a batch exceeds 400 KB.

* The total request size exceeds 16 MB

* Primary key attributes specified on an item in the request do not match
those in the corresponding table's primary key schema.

* You try to perform multiple operations on the same item in the same
BatchWriteItem request. For example, you cannot put and delete the same
item in the same BatchWriteItem request.

* Your request contains at least two items with identical hash and range
keys (which essentially is two put operations)`
