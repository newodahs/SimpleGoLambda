package util

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DymamoSchema interface {
	GetAttrDefs() []types.AttributeDefinition
	GetKeySchema() []types.KeySchemaElement
}

// checks that tableName exists in our dynamodb instance; sets it up if it does not
func EnsureDynamoDBTable(ctx context.Context, cli *dynamodb.Client, tableName string, schemaDef DymamoSchema) error {
	if cli == nil {
		return errors.New("passed dynamodb client was nil")
	}

	if schemaDef == nil {
		return errors.New("nil schema definition passed")
	}

	// look for the table in the dynamodb instance
	if _, err := cli.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}); err != nil {
		var notFoundEx *types.ResourceNotFoundException
		if errors.As(err, &notFoundEx) {
			// not found, create it
			if _, createErr := cli.CreateTable(ctx, &dynamodb.CreateTableInput{
				AttributeDefinitions: schemaDef.GetAttrDefs(),
				KeySchema:            schemaDef.GetKeySchema(),
				TableName:            aws.String(tableName),
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(10),
					WriteCapacityUnits: aws.Int64(10),
				},
			}); createErr != nil {
				return fmt.Errorf("failed to create dynamodb table: %s", createErr)
			}
		} else {
			// something else bad happened
			return fmt.Errorf("failed to validate table [%s] exists in dynamodb: %s", tableName, err)
		}
	}

	return nil
}
