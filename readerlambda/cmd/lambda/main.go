package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/newodahs/readerlambda/pkg/credparser"
	"github.com/newodahs/readerlambda/pkg/util"
)

var (
	initSetup    sync.Once
	_s3Client    *s3.Client
	_dynDBClient *dynamodb.Client
)

// most of this is boiler plate for getting data out of an s3 trigger
func handleRequest(ctx context.Context, s3Event events.S3Event) error {
	log.Printf("lambda init")
	initSetup.Do(func() { // stand up our basic configuration items; should only need to do this once when the lambda starts
		sdkConfig, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatalf("failed to load default config: %s", err)
		}

		_s3Client = s3.NewFromConfig(sdkConfig)
		_dynDBClient = dynamodb.NewFromConfig(sdkConfig)
	})

	if _s3Client == nil || _dynDBClient == nil {
		return fmt.Errorf("s3 client (%v) or dynamodb client (%v) was nil", _s3Client, _dynDBClient)
	}

	// rip over the event records and process the objects
	for _, record := range s3Event.Records {
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.URLDecodedKey

		// get the object
		output, getErr := _s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		})
		if getErr != nil {
			log.Printf("error getting object %s/%s: %s", bucket, key, getErr)
			return getErr
		}
		defer output.Body.Close()

		// parse the data from the s3 object...
		credList, parseErr := credparser.GetCredentialInfo(bufio.NewScanner(output.Body))
		if parseErr != nil {
			if errors.Is(parseErr, credparser.ErrBadParameter) { // die if an unexpected error
				return parseErr
			} else { // rest are really just warnings; should probably refactor to be a bit more sane..
				var wrapErr error
				for ; wrapErr != nil; wrapErr = errors.Unwrap(parseErr) {
					log.Printf("%s", wrapErr)
				}
				log.Printf("%s", parseErr)
			}
		}

		// make sure our dynamodb is basically setup
		if setupErr := util.EnsureDynamoDBTable(ctx, _dynDBClient, `exploitedCredentials`, credparser.CredentialInfo{}); setupErr != nil {
			return setupErr
		}

		// rip over our creds and store
		for _, cred := range credList {
			if _, putErr := _dynDBClient.PutItem(ctx, &dynamodb.PutItemInput{
				Item:      cred.GetKey(),
				TableName: aws.String(`exploitedCredentials`),
			}); putErr != nil {
				log.Printf("WARNING: failed to store exploited credential [%s] to dynamodb: %s", cred, putErr)
			}
		}

		// cleanup the bucket (remove the object we just processed for ease of use)
		if _, delErr := _s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: &bucket,
			Key:    &key,
		}); delErr != nil {
			return delErr
		}
	}

	return nil
}

func main() {
	lambda.Start(handleRequest)
}
