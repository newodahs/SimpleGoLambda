package main

import (
	"context"
	"log"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	apiengine "github.com/newodahs/accessapi/internal/engine"
)

var (
	initSetup sync.Once
	apiEng    *apiengine.APIEngine
	engLambda *ginadapter.GinLambda
)

// most of this is boiler plate for a gin-gonic approach with api-gateway
// basically, instead of using Run/RunTLS/Serve/Whatever, we pass the gin-gonic
// server to a proxy call implemented by the aws-lambda-go-api-proxy project and it makes
// sure the proxied request for the lambda is transformed into something gin-gonic can process
// as a route
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	initSetup.Do(func() {
		gin.SetMode(gin.ReleaseMode)
		apiEng = apiengine.NewAPIEngine("", "", false)
		if apiEng == nil {
			log.Fatal("could not create api engine")
		}
		engLambda = ginadapter.New(apiEng.Server)
	})

	return engLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
