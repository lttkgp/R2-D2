package facebook

import (
	"fmt"
	"log"
	"os"

	fb "github.com/huandu/facebook/v2"
	"github.com/lttkgp/R2-D2/internal/mongo"
	"github.com/lttkgp/R2-D2/internal/utils"
)

const fbGroupID = "1488511748129645"
const mongoDbName = "lttkgp"
const mongoCollectionName = "feed"

var fbFeedParams = fb.Params{"fields": `
id,created_time,from,link,message,message_tags,name,object_id,permalink_url,properties,
shares,source,status_type,type,updated_time,reactions.summary(true){id,name,type},
comments.summary(true){id,attachment,comment_count,created_time,from,like_count,message,message_tags,parent}`}

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

func insertPosts(paging *fb.PagingResult) {
	// Initialize Mongo client
	mongoClient, ctx, cancel, err := mongo.GetMongoClient()
	defer func() {
		cancel()
		if err = mongoClient.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	collection := mongoClient.Database(mongoDbName).Collection(mongoCollectionName)

	// Iterate through page results
	for {
		// Iterate through posts in page
		for _, post := range paging.Data() {
			mongoRes, err := collection.InsertOne(ctx, post)
			if err != nil {
				log.Fatalln(err)
			}
			log.Println(mongoRes.InsertedID)
		}

		// Break on last page
		noMore, err := paging.Next()
		if err != nil {
			panic(err)
		}
		if noMore {
			break
		}
	}
}

// BootstrapDb Bootstrap MongoDB with Facebook posts
func BootstrapDb() {
	fbSession := getFacebookSession()
	feedResp, err := fbSession.Get(fmt.Sprintf("%s/feed", fbGroupID), fbFeedParams)
	if err != nil {
		log.Fatalln(err)
	}
	paging, _ := feedResp.Paging(fbSession)
	insertPosts(paging)
}
