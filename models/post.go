package post

import "gopkg.in/mgo.v2/bson"

type Post struct {
	ID           bson.ObjectId `bson:"_id" json:"id"`
	GraphID      string        `bson:"id" json:"graph_id"`
	CreatedTime  string        `bson:"created_time" json:"created_time"`
	Link         string        `bson:"link" json:"link"`
	Message      string        `bson:"message" json:"message"`
	Name         string        `bson:"name" json:"name"`
	PermalinkURL string        `bson:"permalink_url" json:"permalink_url"`
	Source       string        `bson:"source" json:"source"`
	PostType     string        `bson:"type" json:"post_type"`
	UpdatedTime  string        `bson:"updated_time" json:"updated_time"`
	Reactions    Reaction      `bson:"reactions" json:"reactions"`
	Comments     Comment       `bson:"comments" json:"comments"`
}

type Reaction struct {
	Data    []string         `bson:"data" json:"data"`
	Paging  Cursor           `bson:"paging" json:"paging"`
	Summary ReactionsSummary `bson:"summary" json:"summary"`
}

type Comment struct {
	// Data    []CommentData   `bson:"data" json:"data"`
	Paging  Cursor          `bson:"paging" json:"paging"`
	Summary CommentsSummary `bson:"summary" json:"summary"`
}

type Cursor struct {
	Before string `bson:"before" json:"before"`
	After  string `bson:"after" json:"after"`
}

type ReactionsSummary struct {
	TotalCount     int32 `bson:"total_count" json:"total_count"`
	ViewerReaction int32 `bson:"viewer_reaction" json:"viewer_reaction"`
}

type CommentsSummary struct {
	Order      string `bson:"order" json:"order"`
	TotalCount int32  `bson:"total_count" json:"total_count"`
	CanComment bool   `bson:"can_comment" json:"can_comment"`
}
