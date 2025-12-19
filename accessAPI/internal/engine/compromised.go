package apiengine

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/newodahs/readerlambda/pkg/credparser"
)

const DYNDB_TABLE_EXPLOITCRED = `exploitedCredentials`

// main function for finding compromised accounts via a filter on email or domain
// will fail if no filter is passed
//
// returns a list of the compromised credentials found
func (ae *APIEngine) GetCompromised(c *gin.Context) {
	if ae == nil {
		log.Printf("nil engine called for GetCompromised")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "engine setup failure"})
		return
	}

	if ae.DynDBCli == nil {
		log.Printf("nil dynamodb engine in GetAllCompromised, cannot proceed")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "engine setup failure: no dynamodb client"})
		return
	}

	rawFilter := c.Query("filter") // we'll determine if this is an email or domain on the fly
	if rawFilter == "" {
		log.Printf("empty filter passed to GetCompromised")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid request; must pass a filter for query"})
		return
	}

	var exprErr error
	var expr expression.Expression

	//simple check to see if the filter is for email or domain
	idx := strings.Index(rawFilter, `@`)
	if idx < 0 { // treat this as a domain
		keyEx := expression.Key("domainname").Equal(expression.Value(rawFilter))
		expr, exprErr = expression.NewBuilder().WithKeyCondition(keyEx).Build()
		if exprErr != nil {
			log.Printf("failed to build query expression for dynamodb: %s", exprErr)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "failed to build dynamodb query expression"})
			return
		}
	} else { // it's an email (we hope)
		username := rawFilter[:idx]
		domain := rawFilter[idx+1:]

		keyEx := expression.Key("domainname").Equal(expression.Value(domain)).And(expression.Key("username").Equal(expression.Value(username)))
		expr, exprErr = expression.NewBuilder().WithKeyCondition(keyEx).Build()
		if exprErr != nil {
			log.Printf("failed to build query expression for dynamodb in GetCompromised: %s", exprErr)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "failed to build dynamodb query expression"})
			return
		}
	}

	// query dynamodb and get back the data
	res, getErr := ae.DynDBCli.Query(c.Request.Context(), &dynamodb.QueryInput{
		TableName:                 aws.String(DYNDB_TABLE_EXPLOITCRED),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	})
	if getErr != nil {
		log.Printf("failed during Query call on dynamodb in GetCompromised: %s", getErr)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "failed during Query call on dynamodb"})
		return
	}

	// process the dynamodb data; rip over it and generate an (IMHO) easier-to-use structure that we can return to our client
	errCount := 0
	var output []*credparser.CredentialInfo
	for _, item := range res.Items {
		cred := &credparser.CredentialInfo{}
		unmarshErr := attributevalue.UnmarshalMap(item, cred)
		if unmarshErr != nil {
			log.Printf("failed to unmarshal credential [%+v] in GetCompromised: %s", item, unmarshErr)
			errCount++
		}
		output = append(output, cred)
	}

	c.JSON(http.StatusOK, gin.H{"errorCount": errCount, "credlist": output})
}

// another route implementation I made for just pulling all credentials; not currently exposed but maybe useful
func (ae *APIEngine) GetAllCompromised(c *gin.Context) {
	if ae == nil {
		log.Printf("nil engine called for GetAllCompromised")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "engine setup failure"})
		return
	}

	if ae.DynDBCli == nil {
		log.Printf("nil dynamodb engine in GetAllCompromised, cannot proceed")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "engine setup failure: no dynamodb client"})
		return
	}

	// just pull in everything...
	res, err := ae.DynDBCli.Scan(c.Request.Context(), &dynamodb.ScanInput{
		TableName: aws.String(DYNDB_TABLE_EXPLOITCRED),
	})
	if err != nil {
		log.Printf("failed while attempting to get all compromised account results: %s", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("failed while attempting to get all compromised account results: %s", err)})
		return
	}

	// process the dynamodb data; rip over it and generate an (IMHO) easier-to-use structure that we can return to our client
	errCount := 0
	var output []*credparser.CredentialInfo
	for _, item := range res.Items {
		cred := &credparser.CredentialInfo{}
		unmarshErr := attributevalue.UnmarshalMap(item, cred)
		if unmarshErr != nil {
			log.Printf("failed to unmarshal credential [%+v]: %s", item, unmarshErr)
			errCount++
		}
		output = append(output, cred)
	}

	c.JSON(http.StatusOK, gin.H{"errorCount": errCount, "credlist": output})
}
