package db

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func newFindOptions(batch int) *options.FindOptions {
	return options.Find().SetLimit(int64(batch)).SetSort(bson.D{{Key: "created_at", Value: 1}})
}
