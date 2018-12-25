package dao

import (
	"log"
	"strings"
	"time"

	. "github.com/lttkgp/R2-D2/models"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type TrendingDao struct {
	Server   string
	Database string
}

var db *mgo.Database

const (
	COLLECTION = "posts"
)

func (m *TrendingDao) Connect() {
	session, err := mgo.Dial(m.Server)
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB(m.Database)
}

func (m *TrendingDao) GetTrendingForPeriod(period string) ([]Post, error) {
	var latestPosts []Post
	loc, _ := time.LoadLocation("Europe/London")
	timeNow := time.Now()
	timeUntil := timeNow
	if period == "day" {
		timeUntil = timeNow.AddDate(0, 0, -1)
	} else if period == "month" {
		timeUntil = timeNow.AddDate(0, -1, 0)
	}
	timeUntilString := strings.TrimSuffix(timeUntil.In(loc).Format(time.RFC3339), "Z")
	latestPostsQuery := db.C(COLLECTION).Find(bson.M{"created_time": bson.M{"$gt": timeUntilString}}).Limit(50).Sort("-reactions.summary.total_count").Iter()
	err := latestPostsQuery.All(&latestPosts)
	return latestPosts, err
}

func (m *TrendingDao) GetLatestByCount(count int) ([]Post, error) {
	var trendingPosts []Post
	trendingPostsQuery := db.C(COLLECTION).Find(nil).Sort("-created_time").Limit(count).Iter()
	err := trendingPostsQuery.All(&trendingPosts)
	return trendingPosts, err
}
