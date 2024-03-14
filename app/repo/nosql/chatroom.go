package nosql

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
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

func GetChatroomNoSQLRepoer(ctx context.Context) (repo.ChatroomNoSQLRepoer, error) {
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

func (m MongoRepo) CreateChatroom(ctx context.Context, name string, addUsers []string) error {

	chatroomUUID := uuid.New()

	insertDoc := models.NoSQLChatroomName{
		Name:       name,
		ChatroomID: models.GoUUIDToMongoUUID(chatroomUUID),
	}

	chatroomNameCollection := m.client.Database(database, nil).Collection("chatroom_name")

	for {
		_, err := chatroomNameCollection.InsertOne(
			ctx,
			insertDoc,
		)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				if strings.Contains(err.Error(), "name") {
					return errors.Wrapf(err, "failed to create chatroom, name \"%s\" already taken", name)
				}
			}

			continue
		}

		break
	}

	return nil
}

// Update chatroom with add remove user IDs.
func (m MongoRepo) UpdateChatroom(ctx context.Context, name string, addUsers []string, removeUsers []string) error {
	return errors.New("not implemented yet")
}

// Delete chatroom.
func (m MongoRepo) DeleteChatroom(ctx context.Context, name string) error {
	return errors.New("not implemented yet")
}

// Delete chatroom.
func (m MongoRepo) GetChatroomUUID(ctx context.Context, name string) (uuid.UUID, error) {
	var chatroomID uuid.UUID

	collection := m.client.Database(database, nil).Collection("chatroom_name")

	filter := bson.D{{
		Key:   "name",
		Value: name,
	}}
	var chatroomName models.NoSQLChatroomName
	if err := collection.FindOne(ctx, filter).Decode(&chatroomName); err != nil {
		return chatroomID, errors.Wrapf(err, "failed to find chatroom with name: %s", name)
	}

	return models.MongoUUIDToGoUUID(chatroomName.ChatroomID), nil
}
