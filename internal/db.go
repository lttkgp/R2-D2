package main

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"go.uber.org/zap"
)

// Constants
/// Feed table
var tableName = "feed"
var partitionKey = "facebook_id"
var sortKey = "created_time"

/// GSI
var parsedGsiIndexName = "parsed_index"
var parsedGsiSortKey = "is_parsed"

func createDynamoSession() *dynamodb.DynamoDB {
	dynamoEndpoint := GetEnv("DYNAMODB_ENDPOINT", "")
	if dynamoEndpoint != "" {
		return dynamodb.New(session.Must(session.NewSession(&aws.Config{
			Endpoint: aws.String(dynamoEndpoint),
		})))
	}
	return dynamodb.New(session.Must(session.NewSession()))
}

func createTable(dynamoSession *dynamodb.DynamoDB, logger *zap.Logger) error {
	tableCreateInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String(partitionKey),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String(sortKey),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String(parsedGsiSortKey),
				AttributeType: aws.String("S"),
			},
		},
		BillingMode: aws.String("PROVISIONED"),
		GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
			{
				IndexName: aws.String(parsedGsiIndexName),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String(partitionKey),
						KeyType:       aws.String("HASH"),
					},
					{
						AttributeName: aws.String(parsedGsiSortKey),
						KeyType:       aws.String("RANGE"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String("ALL"),
				},
				ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(1),
					WriteCapacityUnits: aws.Int64(1),
				},
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(partitionKey),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String(sortKey),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		TableName: aws.String(tableName),
	}
	_, err := dynamoSession.CreateTable(&tableCreateInput)
	if err != nil {
		logger.Error("Failed creating table", zap.Error(err))
		return err
	}
	return nil
}

// InitializeDynamoSession creates a DynamoDB session
func InitializeDynamoSession(logger *zap.Logger) (*dynamodb.DynamoDB, error) {
	dynamoSession := createDynamoSession()
	_, err := dynamoSession.DescribeTable(&dynamodb.DescribeTableInput{TableName: aws.String(tableName)})
	if err != nil {
		var resourceNotFoundException *dynamodb.ResourceNotFoundException
		if errors.As(err, &resourceNotFoundException) {
			logger.Warn("Table doesn't exist. Creating...")
			createTableErr := createTable(dynamoSession, logger)
			if createTableErr != nil {
				return nil, createTableErr
			}
		}
		return nil, err
	}
	return dynamoSession, nil
}

func marshalMapWithEmptyCollections(in interface{}) (map[string]*dynamodb.AttributeValue, error) {
	dynamoEncoder := dynamodbattribute.NewEncoder(func(e *dynamodbattribute.Encoder) {
		e.EnableEmptyCollections = true
	})
	av, err := dynamoEncoder.Encode(in)
	if err != nil || av == nil || av.M == nil {
		return map[string]*dynamodb.AttributeValue{}, err
	}
	return av.M, nil
}

func unmarshalMapWithEmptyCollections(m map[string]*dynamodb.AttributeValue, out interface{}) error {
	dynamoDecoder := dynamodbattribute.NewDecoder(func(e *dynamodbattribute.Decoder) {
		e.EnableEmptyCollections = true
	})
	return dynamoDecoder.Decode(&dynamodb.AttributeValue{M: m}, out)
}

// UpdateOrInsertPost updates a post by ID, or creates it if it doesn't already exist.
func UpdateOrInsertPost(dynamoSession *dynamodb.DynamoDB, postData PostData, logger *zap.Logger) {
	// Using a custom marshal method since comments & reaction summary have an empty object value
	marshalledPostData, err := marshalMapWithEmptyCollections(postData.FacebookPost)
	if err != nil {
		logger.Error("Unable to marshal Facebook post", zap.Error(err))
	}

	createdTime := postData.CreatedTime.Format(time.RFC3339)
	key := map[string]*dynamodb.AttributeValue{
		partitionKey: {S: &postData.FacebookID},
		sortKey:      {S: &createdTime},
	}
	expressionAttributeNames := map[string]*string{
		"#I": &parsedGsiSortKey,
		"#P": aws.String("post"),
	}
	expressionAttributeValues := map[string]*dynamodb.AttributeValue{
		":I": {S: aws.String("false")},
		":P": {M: marshalledPostData},
	}
	updateItemInput := dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
		Key:                       key,
		TableName:                 &tableName,
		UpdateExpression:          aws.String("SET #P = if_not_exists(#P, :P), #I = :I"),
	}
	_, err = dynamoSession.UpdateItem(&updateItemInput)
	if err != nil {
		logger.Error("Failed to UpdateOrInsertPost", zap.String("FacebookId", postData.FacebookID), zap.Error(err))
	}
	logger.Info("UpdateOrInsertPost success", zap.String("FacebookID", postData.FacebookID))
}

// MarkPostAsParsed marks a post in DB as parsed by C-3PO
func MarkPostAsParsed(dynamoSession *dynamodb.DynamoDB, postData PostData, logger *zap.Logger) bool {
	key := map[string]*dynamodb.AttributeValue{
		partitionKey: {S: &postData.FacebookID},
		sortKey:      {S: aws.String(postData.CreatedTime.Format(time.RFC3339))},
	}
	expressionAttributeNames := map[string]*string{
		"#I": &parsedGsiSortKey,
	}

	updateItemInput := dynamodb.UpdateItemInput{
		ExpressionAttributeNames: expressionAttributeNames,
		Key:                      key,
		TableName:                &tableName,
		UpdateExpression:         aws.String("REMOVE #I"),
	}
	_, err := dynamoSession.UpdateItem(&updateItemInput)
	if err != nil {
		logger.Warn("MarkPostAsParsed failed", zap.Error(err))
		return false
	}
	return true
}
