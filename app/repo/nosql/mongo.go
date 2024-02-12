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

func (m MongoRepo) GetChatroomLogs(ctx context.Context, params models.GetDBMessagesParams) ([]models.ChatroomLog, error) {
	collection := m.client.Database(database, nil).Collection("chat_logs")

	opts := options.Find().SetSort(bson.D{{"timestamp", -1}})
	cursor, err := collection.Find(ctx, bson.D{{"chatroom_id", params.ChatroomID.String()}}, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find chatroom logs")
	}

	var results []models.ChatroomLog
	if err = cursor.All(ctx, &results); err != nil {
		return nil, errors.Wrap(err, "failed to read logs from cursor")
	}

	return results, nil

	// chatroomLogs := []models.ChatroomLog{}
	// for _, result := range results {
	// 	var chatroomLog models.ChatroomLog
	// 	bson.Unmarshal(result, &chatroomLog)
	// }
}

func (m MongoRepo) SetChatroomLogs(ctx context.Context, params models.SetDBMessagesParams) error {
	collection := m.client.Database(database, nil).Collection("chat_logs")

	collection.InsertOne(ctx, models.ChatroomLog{
		ChatroomID: params.ChatroomID,
		ClientID:   params.ClientID,
		Text:       params.Text,
		Timestamp:  params.Timestamp,
	})
	return errors.New("not yet implemented")
}
