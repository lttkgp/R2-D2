package song

import "gopkg.in/mgo.v2/bson"

// Song model
type Song struct {
	ID         bson.ObjectId `bson:"_id" json:"id"`
	AlbumArt   string        `bson:"album_art" json:"album_art"`
	SongName   string        `bson:"name" json:"name"`
	ArtistName string        `bson:"artist_name" json:"artist_name"`
}
