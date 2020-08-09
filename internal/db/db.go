package db

import (
	"context"
	"time"

	"github.com/lttkgp/R2-D2/internal/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetMongoClient Get a Mongo Client with correct DB URI
func GetMongoClient() (*mongo.Client, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 72*time.Hour)
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(utils.GetEnv("MONGO_URI", "")))
	return mongoClient, ctx, cancel, err
}
