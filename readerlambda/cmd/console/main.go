package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	transport "github.com/aws/smithy-go/endpoints"
	"github.com/newodahs/readerlambda/pkg/credparser"
	"github.com/newodahs/readerlambda/pkg/util"
)

type Resolver struct {
	URL *url.URL
}

func (r *Resolver) ResolveEndpoint(_ context.Context, params dynamodb.EndpointParameters) (transport.Endpoint, error) {
	u := *r.URL
	return transport.Endpoint{URI: u}, nil
}

func main() {
	credFile := flag.String(`credfile`, `./test/challenge_creds.txt`, `Pass the name of the file where the credentials to be read are stored`)
	localDynamo := flag.Bool(`localdb`, false, `If set, will attempt to write to a local dynamodb instance`)
	flag.Parse()

	if credFile == nil {
		flag.Usage()
		os.Exit(1)
	}

	//parse the credentials file
	credList, err := credparser.GetCredentialInfoFile(*credFile)
	if err != nil {
		var wrapErr error
		for ; wrapErr != nil; wrapErr = errors.Unwrap(err) {
			log.Printf("%s", wrapErr)
		}
		log.Printf("%s", err)
	}

	//if set, ensure the local dynamodb instance is accessable
	var cli *dynamodb.Client
	if *localDynamo {
		log.Printf("Writing credential data to local dynamodb...")

		dynamoDBURL, parseErr := url.Parse("http://localhost:8000")
		if parseErr != nil {
			log.Fatalf("failed to parse local dynamodb url: %s", parseErr)
		}

		cli = dynamodb.New(dynamodb.Options{
			EndpointResolverV2: &Resolver{URL: dynamoDBURL},
			Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     `fakeMyKeyId`,
					SecretAccessKey: `fakeSecretAccessKey`,
				}, nil
			}),
			Region: `fakeRegion`,
		})

		if setupErr := util.EnsureDynamoDBTable(context.TODO(), cli, `exploitedCredentials`, credparser.CredentialInfo{}); setupErr != nil {
			log.Printf("failed to setup exploitedCredentials table in local dynamodb: %s", setupErr)
		}
	}

	for _, cred := range credList {
		// for our own sanity (in test), print out what we parsed
		fmt.Printf("User: %s; Domain: %s; Email: %s; Password: %s\n", cred.User, cred.Domain, cred.Email, cred.Password)

		// if set, dump the parse cred to the local dynamodb instance
		if *localDynamo && cli != nil {
			if _, putErr := cli.PutItem(context.TODO(), &dynamodb.PutItemInput{
				Item:      cred.GetKey(),
				TableName: aws.String(`exploitedCredentials`),
			}); putErr != nil {
				log.Printf("failed to store exploited credential [%s] to dynamodb: %s", cred, putErr)
			}
		}
	}
}
