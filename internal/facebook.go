package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	fb "github.com/huandu/facebook/v2"
)

const fbGroupID = "1488511748129645"

var whoamiHeaderVal = GetEnv("WHOAMI", "")
var fbFeedParams = fb.Params{
	"fields": `
id,created_time,from,link,message,message_tags,name,object_id,permalink_url,properties,
shares,source,status_type,type,updated_time,reactions.summary(true){id,name,type},
comments.summary(true){id,attachment,comment_count,created_time,from,like_count,message,message_tags,parent}`,
	"limit": "100",
}

func retryNotifyFunc(err error, duration time.Duration) {
	log.Println(fmt.Sprintf("Queued for retry after %s, error=%s", duration, err))
}

func getFbAccessToken(fbApp *fb.App) string {
	longAccessToken := GetEnv("FB_LONG_ACCESS_TOKEN", "")
	if longAccessToken == "" {
		shortAccessToken := GetEnv("FB_SHORT_ACCESS_TOKEN", "")
		if shortAccessToken == "" {
			return shortAccessToken
		}
		var err error
		longAccessToken, _, err = fbApp.ExchangeToken(shortAccessToken)

		// Update env
		UpdateEnvFile("FB_LONG_ACCESS_TOKEN", longAccessToken)
		setEnvErr := os.Setenv("FB_LONG_ACCESS_TOKEN", longAccessToken)
		if err != nil || setEnvErr != nil {
			return ""
		}
	}
	return longAccessToken
}

func getFacebookSession() *fb.Session {
	var fbApp = fb.New(GetEnv("FB_APP_ID", ""), GetEnv("FB_APP_SECRET", ""))
	fbApp.RedirectUri = "https://beta.lttkgp.com"
	fbSession := fbApp.Session(getFbAccessToken(fbApp))
	fbSession.RFC3339Timestamps = true

	return fbSession
}

// BootstrapDb bootstraps the DB with Facebook posts
func BootstrapDb() {
	// Initialize Facebook session
	fbSession := getFacebookSession()
	fbSession.Version = "v8.0"

	// Initialize Database session
	dynamoSession := CreateDynamoSession()

	// Configure exponential backoff for retries
	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.MaxInterval = 24 * time.Hour

	// Fetch the first page of response
	var feedResp fb.Result
	err := backoff.RetryNotify(func() error {
		var fbError error
		feedResp, fbError = fbSession.Get(fmt.Sprintf("%s/feed", fbGroupID), fbFeedParams)
		return fbError
	}, exponentialBackoff, retryNotifyFunc)
	if err != nil {
		log.Fatalln(err)
	}
	paging, _ := feedResp.Paging(fbSession)

	// Iterate through page results
	for {
		// Iterate through posts in page
		for _, post := range paging.Data() {
			// Read keys
			var keyMetadata KeyMetadata
			err := post.Decode(&keyMetadata)
			if err != nil {
				log.Fatalln(err)
			}
			log.Println(keyMetadata)

			postData := PostData{
				CreatedTime:  keyMetadata.CreatedTime,
				FacebookID:   keyMetadata.FacebookID,
				FacebookPost: post,
				IsParsed:     "false",
			}

			// Insert post to DB
			UpdateOrInsertPost(dynamoSession, postData)
		}

		// Break on last page
		var noMore bool
		err := backoff.RetryNotify(func() error {
			var fbError error
			noMore, fbError = paging.Next()
			return fbError
		}, exponentialBackoff, retryNotifyFunc)
		if err != nil {
			panic(err)
		}
		if noMore {
			break
		}
	}
}
