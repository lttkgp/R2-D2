package dao

import (
	"log"

	. "github.com/lttkgp/R2-D2/models"
	mgo "gopkg.in/mgo.v2"
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

func (m *TrendingDao) GetTrendingForPeriod(id string) ([]Post, error) {
	var trendingPosts []Post
	trendingPostsQuery := db.C(COLLECTION).Find(nil).Sort("-created_time").Limit(10).Iter()
	err := trendingPostsQuery.All(&trendingPosts)
	return trendingPosts, err
}
