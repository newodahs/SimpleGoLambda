package apiengine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	transport "github.com/aws/smithy-go/endpoints"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// wrap up some common items that our routes may need
type APIEngine struct {
	Server      *gin.Engine
	SSLCertFile string
	SSLKeyFile  string
	DynDBCli    *dynamodb.Client
}

// Really only useful for our local test harness runs; the lambda uses a Proxy call and not this...
func (ae APIEngine) Run(bindAddr string, bindPort uint16) error {
	if bindAddr == "" {
		bindAddr = "0.0.0.0"
	}

	if ae.SSLCertFile != "" && ae.SSLKeyFile != "" { // we determine our run mode by if we have an SSL cert/key or not; WARNING: Not Fully Implemented/Tested
		if bindPort == 0 {
			bindPort = 443
		}
		return ae.Server.RunTLS(fmt.Sprintf("%s:%d", bindAddr, bindPort), ae.SSLCertFile, ae.SSLKeyFile)
	}

	log.Printf("WARNING: No SSL Cert and Key file were provided, running in HTTP (non-SSL) mode!")
	if bindPort == 0 {
		bindPort = 80
	}

	return ae.Server.Run(fmt.Sprintf("%s:%d", bindAddr, bindPort))
}

// leave localResolver nil if not going to a local dynamodb localResolver dynamodb.EndpointResolverV2
func NewAPIEngine(sslCertFile, sslKeyFile string, useLocalDynamoDB bool) *APIEngine {
	ret := &APIEngine{SSLCertFile: sslCertFile, SSLKeyFile: sslKeyFile}
	ret.Server = gin.Default()
	if trustErr := ret.Server.SetTrustedProxies(nil); trustErr != nil {
		log.Printf("failed to set trusted proxies to off (will continue): %s", trustErr)
	}

	//no going crazy here, this is just a demo
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Role", "Authorization"}
	ret.Server.Use(cors.New(corsConfig))

	//TODO: better error handling...
	if err := ret.setupRoutes(); err != nil {
		log.Fatalf("failed to setup gin-gonic routes: %s", err)
		return nil
	}

	//TODO: better error handling...
	if err := ret.setupDynamoDB(useLocalDynamoDB); err != nil {
		log.Fatalf("failed to setup dynamodb: %s", err)
		return nil
	}

	return ret
}

// Resolver is used for local dynamodb instances only
type Resolver struct {
	URL *url.URL
}

func (r *Resolver) ResolveEndpoint(_ context.Context, params dynamodb.EndpointParameters) (transport.Endpoint, error) {
	u := *r.URL
	return transport.Endpoint{URI: u}, nil
}

func (ae *APIEngine) setupDynamoDB(useLocalDynamoDB bool) error {
	if ae == nil {
		return errors.New("nil gin-engine passed to setupDynamoDB")
	}

	// HACK: if we're in our test harness, force our dynamodb client to look locally
	if useLocalDynamoDB {
		dynamoURL, parseErr := url.Parse("http://localhost:8000")
		if parseErr != nil {
			return fmt.Errorf("failed to parse dynamodb local url: %s", parseErr)
		}

		ae.DynDBCli = dynamodb.New(dynamodb.Options{
			EndpointResolverV2: &Resolver{URL: dynamoURL},
			Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     `fakeMyKeyId`,
					SecretAccessKey: `fakeSecretAccessKey`,
				}, nil
			}),
			Region: `fakeRegion`,
		})
	} else { //normal path in AWS deployment
		sdkConfig, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to load default config: %s", err)
		}

		ae.DynDBCli = dynamodb.NewFromConfig(sdkConfig)
	}

	return nil
}

// route setup
func (ae *APIEngine) setupRoutes() error {
	if ae == nil {
		return errors.New("nil gin-engine passed to setupRoutes")
	}

	// only one group for this api, keep it simple
	versionGrp := ae.Server.Group("/v1")
	{
		versionGrp.GET("/compromised", ae.GetCompromised) // actual call to look up compromised creds

		versionGrp.GET("/ping", func(ctx *gin.Context) { // for debug purposes (make sure it's basically working)
			ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}

	return nil
}
