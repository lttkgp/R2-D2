package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	fb "github.com/huandu/facebook/v2"
	"github.com/lttkgp/R2-D2/internal/db"
	"github.com/lttkgp/R2-D2/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const fbGroupID = "1488511748129645"
const mongoDbName = "lttkgp"
const feedCollectionName = "feed"

var fbFeedParams = fb.Params{"fields": `
id,created_time,from,link,message,message_tags,name,object_id,permalink_url,properties,
shares,source,status_type,type,updated_time,reactions.summary(true){id,name,type},
comments.summary(true){id,attachment,comment_count,created_time,from,like_count,message,message_tags,parent}`}

// MongoPost describes the data structure to be inserted into Mongo
type MongoPost struct {
	IsParsed     bool      `bson:"is_parsed" json:"is_parsed"`
	FacebookID   string    `bson:"facebook_id" json:"facebook_id"`
	FacebookPost fb.Result `bson:"facebook_post" json:"facebook_post"`
}

func retryNotifyFunc(err error, duration time.Duration) {
	log.Println(fmt.Sprintf("Queued for retry after %s, error=%s", duration, err))
}

func getFbAccessToken(fbApp *fb.App) string {
	longAccessToken := utils.GetEnv("FB_LONG_ACCESS_TOKEN", "")
	if longAccessToken == "" {
		shortAccessToken := utils.GetEnv("FB_SHORT_ACCESS_TOKEN", "")
		if shortAccessToken == "" {
			return shortAccessToken
		}
		var err error
		longAccessToken, _, err = fbApp.ExchangeToken(shortAccessToken)
		setEnvErr := os.Setenv("FB_LONG_ACCESS_TOKEN", longAccessToken)
		if err != nil || setEnvErr != nil {
			return ""
		}
	}
	return longAccessToken
}

func getFacebookSession() *fb.Session {
	var fbApp = fb.New(utils.GetEnv("FB_APP_ID", ""), utils.GetEnv("FB_APP_SECRET", ""))
	fbApp.RedirectUri = "https://beta.lttkgp.com"
	fbSession := fbApp.Session(getFbAccessToken(fbApp))
	fbSession.RFC3339Timestamps = true

	return fbSession
}

func updateOrInsertPost(ctx context.Context, collection *mongo.Collection, mongoPost MongoPost) {
	shouldUpsert := true
	replaceOptions := options.ReplaceOptions{Upsert: &shouldUpsert}
	replaceFilter := bson.M{"facebook_id": mongoPost.FacebookID}
	mongoRes, err := collection.ReplaceOne(ctx, replaceFilter, mongoPost, &replaceOptions)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(mongoRes.UpsertedID)
}

func insertPosts(paging *fb.PagingResult) {
	// Initialize Mongo client
	mongoClient, ctx, cancel, err := db.GetMongoClient()
	defer func() {
		cancel()
		if err = mongoClient.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	feedCollection := mongoClient.Database(mongoDbName).Collection(feedCollectionName)

	// Iterate through page results
	for {
		// Iterate through posts in page
		for _, post := range paging.Data() {
			var facebookID string
			err := post.DecodeField("id", &facebookID)
			if err != nil {
				log.Fatalln(err)
			}
			mongoPost := MongoPost{
				IsParsed:     false,
				FacebookID:   facebookID,
				FacebookPost: post,
			}
			updateOrInsertPost(ctx, feedCollection, mongoPost)
		}

		exponentialBackoff := backoff.NewExponentialBackOff()
		exponentialBackoff.MaxInterval = 6 * time.Hour

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

// BootstrapDb bootstraps MongoDB with Facebook posts
func BootstrapDb() {
	fbSession := getFacebookSession()
	fbSession.Version = "v7.0"

	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.MaxInterval = 6 * time.Hour

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
	insertPosts(paging)
}

// DispatchFreshPosts picks up the posts Mongo which have is_parsed=false and sends them to C3PO
func DispatchFreshPosts() {
	// Initialize Mongo client
	mongoClient, ctx, cancel, err := db.GetMongoClient()
	defer func() {
		cancel()
		if err = mongoClient.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	feedCollection := mongoClient.Database(mongoDbName).Collection(feedCollectionName)

	cur, err := feedCollection.Find(ctx, bson.M{"is_parsed": false})
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		err := cur.Close(ctx)
		if err != nil {
			log.Println(err)
		}
	}()
	for cur.Next(ctx) {
		var result MongoPost
		err := cur.Decode(&result)
		if err != nil {
			log.Fatal(err)
		}

		// POST request to C-3PO
		requestBody, err := json.Marshal(result)
		if err != nil {
			log.Fatalln(err)
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/v1/data/post", utils.GetEnv("C3PO_URI", "")),
			"application/json",
			bytes.NewBuffer(requestBody))
		if err != nil {
			log.Fatalln(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println(string(body))

		err = resp.Body.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
}
