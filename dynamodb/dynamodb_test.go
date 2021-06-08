package dynamodb_test

import (
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
	"github.com/juliaqiuxy/wfcache"
	dynamodbAdapter "github.com/juliaqiuxy/wfcache/dynamodb"
)

var dynamodbOnce sync.Once
var dynamodbInstance *dynamodb.DynamoDB

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

func TestDynamoDb(t *testing.T) {
	dynamodbClient := DynamodbClient()

	c, _ := wfcache.New(
		dynamodbAdapter.Create(dynamodbClient, "tests", 6*time.Hour),
	)

	key := "my_key"
	val := "my_value"

	c.Set(key, val)

	items, err := c.BatchGet([]string{key})

	if err != nil {
		t.Errorf("Expected 1 item, got none %s", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 items, got %v", len(items))
	}

	var str string
	json.Unmarshal(items[0].Value, &str)

	if str != val {
		t.Errorf("Expected %v (type %v), got %v (type %v), ", str, reflect.TypeOf(str), val, reflect.TypeOf(val))
	}

	fmt.Println(items, str, err)
}
