package dynamodb_test

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/juliaqiuxy/wfcache"
	dynamodbAdapter "github.com/juliaqiuxy/wfcache/dynamodb"
)

var dynamodbOnce sync.Once
var dynamodbInstance *dynamodb.DynamoDB

func DynamodbClient() *dynamodb.DynamoDB {
	var newLocalDynamodb = func() *dynamodb.DynamoDB {
		s, err := session.NewSessionWithOptions(session.Options{
			Profile: "dynamodb-local",
			Config: aws.Config{
				Region:      aws.String("dev-region"),
				Endpoint:    aws.String("http://localhost:8000"),
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

func TestDynamoDb(t *testing.T) {
	dynamodbClient := DynamodbClient()

	c, _ := wfcache.Create(
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

	fmt.Println(items, str, err)
}
