package nosql

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepo struct {
	client *mongo.Client
}

const (
	uri      string = "mongodb://mongodb:27017"
	database string = "chatroom"
)

func GetChatroomLogRepoer(ctx context.Context) (repo.ChatroomLogRepoer, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongo instance")
	}

	return MongoRepo{
		client: client,
	}, nil
}

func (m MongoRepo) SelectChatroomLogs(ctx context.Context, params models.SelectDBMessagesParams) ([]models.ChatroomLog, error) {
	collection := m.client.Database(database, nil).Collection("chat_logs")

	filter := bson.D{{
		Key:   "chatroom_id",
		Value: models.GoUUIDToMongoUUID(params.ChatroomID),
	}}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find chatroom logs")
	}

	var results []models.ChatroomLog
	for cursor.Next(ctx) {
		var result models.ChatroomLogMongo
		if err := cursor.Decode(&result); err != nil {
			return nil, errors.Wrap(err, "failed to decode chatroom logs")
		}

		results = append(results, result.ConvertToChatroomLog())
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read logs from cursor")
	}

	return results, nil
}

func (m MongoRepo) InsertChatroomLogs(ctx context.Context, params models.InsertDBMessagesParams) error {
	collection := m.client.Database(database, nil).Collection("chat_logs")

	if _, err := collection.InsertOne(ctx, params.ConvertToChatroomLogMongo()); err != nil {
		return errors.Wrap(err, "failed to insert chatroom message")
	}

	return nil
}
