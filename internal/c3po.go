package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"go.uber.org/zap"
)

// dispatchItem parses and sends a post to C-3PO and marks the post as read if successful
func dispatchItem(dynamoSession *dynamodb.DynamoDB, entry map[string]*dynamodb.AttributeValue, logger *zap.Logger) error {
	// Parse entry
	var postData PostData
	err := unmarshalMapWithEmptyCollections(entry, &postData)
	if err != nil {
		logger.Error("Failed to unmarshal post from DB", zap.Any("entry", entry))
		return err
	}

	// Prepare request
	requestBody, err := json.Marshal(C3poRequest{FacebookPost: postData.FacebookPost})
	if err != nil {
		logger.Error("Failed to marshal DB post to Facebook post", zap.Object("postData", postData), zap.Error(err))
		return err
	}
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1/data/post", GetEnv("C3PO_URI", "")),
		bytes.NewBuffer(requestBody))
	if err != nil {
		logger.Error("Failed to generate post request payload for C-3PO", zap.Object("postData", postData), zap.Error(err))
		return err
	}
	req.Header.Set("whoami", whoamiHeaderVal)
	req.Header.Set("Content-Type", "application/json")

	// Make POST request to C3PO
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("POST request to C-3PO failed", zap.Error(err))
		return err
	}

	// Parse response body as bytes
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Warn("Failed parsing C-3PO response as bytes", zap.Error(err))
		return err
	}

	// Parse response
	var c3poResponse C3poResponse
	err = json.Unmarshal(body, &c3poResponse)
	if err != nil {
		logger.Error("Failed to unmarshal JSON response from C-3PO", zap.Error(err))
		return err
	}
	if c3poResponse.Success {
		markParsedSuccess := MarkPostAsParsed(dynamoSession, postData, logger)
		if markParsedSuccess {
			logger.Info("Successfully parsed", zap.String("postId", postData.FacebookID))
		} else {
			logger.Warn("Failed to parse", zap.String("postId", postData.FacebookID))
		}
	} else {
		logger.Warn("Failed to parse", zap.String("postId", postData.FacebookID))
	}

	err = resp.Body.Close()
	if err != nil {
		logger.Warn("Failed to closed HTTP response body", zap.Error(err))
		return err
	}
	return nil
}

// DispatchFreshPosts picks up the posts which have is_parsed=false and sends them to C3PO
func DispatchFreshPosts(dynamoSession *dynamodb.DynamoDB, logger *zap.Logger) error {
	if whoamiHeaderVal == "" {
		logger.Fatal("C-3PO header env variable `WHOAMI` not present")
	}

	// Fetch all posts which are not yet parsed
	fetchUnparsedPostsQueryNew := dynamodb.ScanInput{
		ExpressionAttributeNames:  map[string]*string{"#isParsed": &parsedGsiSortKey},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":isParsed": {S: aws.String("false")}},
		FilterExpression:          aws.String("#isParsed = :isParsed"),
		IndexName:                 &parsedGsiIndexName,
		TableName:                 &tableName,
	}

	err := dynamoSession.ScanPages(&fetchUnparsedPostsQueryNew, func(output *dynamodb.ScanOutput, b bool) bool {
		for _, entry := range output.Items {
			err := dispatchItem(dynamoSession, entry, logger)
			if err != nil {
				logger.Warn("Dispatching post to C-3PO failed", zap.Error(err))
				continue
			}
		}

		if len(output.LastEvaluatedKey) != 0 {
			fetchUnparsedPostsQueryNew.SetExclusiveStartKey(output.LastEvaluatedKey)
			return true
		}
		return false
	})
	if err != nil {
		logger.Warn("Failed scanning DB pages", zap.Error(err))
		return err
	}

	return nil
}
