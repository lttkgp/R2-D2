package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getMongoClient() (*mongo.Client, *context.Context, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(getEnv("MONGO_URI", "")))
	return mongoClient, &ctx, err
}
