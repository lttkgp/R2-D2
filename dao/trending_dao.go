package dao

import (
	"log"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type TrendingDao struct {
	Server   string
	Database string
}

var db *mgo.Database

const (
	COLLECTION = "songs"
)

func (m *TrendingDao) Connect() {
	session, err := mgo.Dial(m.Server)
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB(m.Database)
}

func (m *TrendingDao) FindById(id string) (Song, error) {
	var song Song
	err := db.C(COLLECTION).FindId(bson.ObjectIdHex(id)).One(&song)
	return song, err
}
