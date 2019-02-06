package dao

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
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
	dialInfo, err0 := mgo.ParseURL(m.Server)
	if err0 != nil {
		log.Fatal(err0)
	}
	fmt.Println(m.Server)
	if m.Server != "mongodb://localhost" {
		tlsConfig := &tls.Config{}
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}
	}
	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to DB!")
	db = session.DB(m.Database)
}

func (m *TrendingDao) GetTrendingForPeriod(period string) ([]Post, error) {
	var latestPosts []Post
	loc, _ := time.LoadLocation("Europe/London")
	timeNow := time.Now()
	timeUntil := timeNow
	if period == "day" {
		timeUntil = timeNow.AddDate(0, 0, -1)
	} else if period == "week" {
		timeUntil = timeNow.AddDate(0, 0, -7)
	} else if period == "month" {
		timeUntil = timeNow.AddDate(0, -1, 0)
	}
	timeUntilString := strings.TrimSuffix(timeUntil.In(loc).Format(time.RFC3339), "Z")
	latestPostsQuery := db.C(COLLECTION).Find(bson.M{"created_time": bson.M{"$gt": timeUntilString}, "type": "video"}).Limit(50).Sort("-reactions.summary.total_count").Iter()
	err := latestPostsQuery.All(&latestPosts)
	return latestPosts, err
}

func (m *TrendingDao) GetLatestByCount(count int) ([]Post, error) {
	var trendingPosts []Post
	trendingPostsQuery := db.C(COLLECTION).Find(bson.M{"type": "video"}).Sort("-created_time").Limit(count).Iter()
	err := trendingPostsQuery.All(&trendingPosts)
	return trendingPosts, err
}
