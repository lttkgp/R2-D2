package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Constants
var tableName = "feed"
var partitionKey = "facebook_id"
var sortKey = "created_time"
var parsedGsiIndexName = "parsed_index"
var parsedGsiPartitionKey = "is_parsed"

// CreateDynamoSession creates a DynamoDB session
func CreateDynamoSession() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSession())
	return dynamodb.New(sess)
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
func UpdateOrInsertPost(dynamoSession *dynamodb.DynamoDB, postData PostData) {
	// Using a custom marshal method since comments & reaction summary have an empty object value
	marshalledPostData, err := marshalMapWithEmptyCollections(postData.FacebookPost)
	if err != nil {
		log.Fatalln(err)
	}

	createdTime := postData.CreatedTime.Format(time.RFC3339)
	key := map[string]*dynamodb.AttributeValue{
		partitionKey: {S: &postData.FacebookID},
		sortKey:      {S: &createdTime},
	}
	expressionAttributeNames := map[string]*string{
		"#I": &parsedGsiPartitionKey,
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
		log.Fatalln(err)
	}
	log.Println(postData.FacebookID)
}

// MarkPostAsParsed marks a post in DB as parsed by C-3PO
func MarkPostAsParsed(dynamoSession *dynamodb.DynamoDB, postData PostData) bool {
	key := map[string]*dynamodb.AttributeValue{
		partitionKey: {S: &postData.FacebookID},
		sortKey:      {S: aws.String(postData.CreatedTime.Format(time.RFC3339))},
	}
	expressionAttributeNames := map[string]*string{
		"#I": &parsedGsiPartitionKey,
	}

	updateItemInput := dynamodb.UpdateItemInput{
		ExpressionAttributeNames: expressionAttributeNames,
		Key:                      key,
		TableName:                &tableName,
		UpdateExpression:         aws.String("REMOVE #I"),
	}
	_, err := dynamoSession.UpdateItem(&updateItemInput)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

// dispatchScanOutput parses and sends posts from a DB scan page to C-3PO and marks the post as read if successful
func dispatchScanOutput(dynamoSession *dynamodb.DynamoDB, output *dynamodb.ScanOutput) {
	for _, entry := range output.Items {
		// Parse entry
		var postData PostData
		err := unmarshalMapWithEmptyCollections(entry, &postData)
		if err != nil {
			log.Fatalln(err)
		}

		// Prepare request
		requestBody, err := json.Marshal(C3poRequest{FacebookPost: postData.FacebookPost})
		if err != nil {
			log.Fatalln(err)
		}
		req, err := http.NewRequest(
			"POST",
			fmt.Sprintf("%s/v1/data/post", GetEnv("C3PO_URI", "")),
			bytes.NewBuffer(requestBody))
		if err != nil {
			log.Fatalln(err)
		}
		req.Header.Set("whoami", whoamiHeaderVal)
		req.Header.Set("Content-Type", "application/json")

		// Make POST request to C3PO
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			resp.Body.Close()
			continue
		}

		// Parse response body as bytes
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			resp.Body.Close()
			continue
		}

		// Parse response
		var c3poResponse C3poResponse
		err = json.Unmarshal(body, &c3poResponse)
		if err != nil {
			log.Fatalln(err)
		}
		if c3poResponse.Success {
			markParsedSuccess := MarkPostAsParsed(dynamoSession, postData)
			if markParsedSuccess {
				log.Println(fmt.Sprintf("Successfully parsed postId=%s", postData.FacebookID))
			} else {
				log.Println(fmt.Sprintf("Failed to parse postId=%s", postData.FacebookID))
			}
		} else {
			log.Println(fmt.Sprintf("Failed to parse postId=%s", postData.FacebookID))
		}

		err = resp.Body.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}
}

// DispatchFreshPosts picks up the posts which have is_parsed=false and sends them to C3PO
func DispatchFreshPosts() {
	if whoamiHeaderVal == "" {
		log.Fatalln("C3PO header env variable `WHOAMI` not present")
	}

	// Create a DynamoDB session
	dynamoSession := CreateDynamoSession()

	// Fetch all posts which are not yet parsed
	fetchUnparsedPostsQueryNew := dynamodb.ScanInput{
		ExpressionAttributeNames:  map[string]*string{"#isParsed": &parsedGsiPartitionKey},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":isParsed": {S: aws.String("false")}},
		FilterExpression:          aws.String("#isParsed = :isParsed"),
		IndexName:                 &parsedGsiIndexName,
		TableName:                 &tableName,
	}

	err := dynamoSession.ScanPages(&fetchUnparsedPostsQueryNew, func(output *dynamodb.ScanOutput, b bool) bool {
		dispatchScanOutput(dynamoSession, output)
		if len(output.LastEvaluatedKey) != 0 {
			fetchUnparsedPostsQueryNew.SetExclusiveStartKey(output.LastEvaluatedKey)
			return true
		} else {
			return false
		}
	})
	if err != nil {
		log.Println(err)
	}
}
