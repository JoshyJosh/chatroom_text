package nosql

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	"context"
	"fmt"
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
		var result models.NoSQLChatroomLog
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

	if _, err := collection.InsertOne(ctx, params.ConvertToNoSQLChatroomLog()); err != nil {
		return errors.Wrap(err, "failed to insert chatroom message")
	}

	return nil
}

func (m MongoRepo) CreateChatroom(ctx context.Context, name string, addUsers []string) (uuid.UUID, error) {

	chatroomUUID := uuid.New()

	insertDoc := models.NoSQLChatroomEntry{
		Name:       name,
		ChatroomID: models.GoUUIDToMongoUUID(chatroomUUID),
		IsActive:   true,
	}

	collection := m.client.Database(database, nil).Collection("chatroom_list")

	for {
		_, err := collection.InsertOne(
			ctx,
			insertDoc,
		)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				if strings.Contains(err.Error(), "name") {
					return chatroomUUID, errors.Wrapf(err, "failed to create chatroom, name \"%s\" already taken", name)
				}

				if strings.Contains(err.Error(), "chatroom_id") {
					return chatroomUUID, errors.Wrapf(err, "failed to create chatroom, chatroom_id \"%s\" already taken", chatroomUUID.String())
				}
			}

			continue
		}

		break
	}

	return chatroomUUID, nil
}

// Update chatroom with add remove user IDs.
func (m MongoRepo) UpdateChatroom(ctx context.Context, chatroomID uuid.UUID, newName string, addUsers []string, removeUsers []string) error {
	collection := m.client.Database(database, nil).Collection("chatroom_list")

	filter := bson.D{{Key: "chatroom_id", Value: models.GoUUIDToMongoUUID(chatroomID)}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: newName}}}}
	res, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.Wrap(err, "failed to delete chatroom")
	}

	if res.MatchedCount == 0 {
		return fmt.Errorf("failed to match chatroom id %s for update", chatroomID)
	}

	return nil
}

// Delete chatroom.
func (m MongoRepo) DeleteChatroom(ctx context.Context, chatroomID uuid.UUID) error {
	collection := m.client.Database(database, nil).Collection("chatroom_list")

	filter := bson.D{{Key: "chatroom_id", Value: models.GoUUIDToMongoUUID(chatroomID)}}
	if _, err := collection.DeleteOne(ctx, filter); err != nil {
		return errors.Wrap(err, "failed to delete chatroom")
	}

	return nil
}

// Delete chatroom.
func (m MongoRepo) GetChatroomEntry(ctx context.Context, chatroomID uuid.UUID) (models.ChatroomEntry, error) {
	collection := m.client.Database(database, nil).Collection("chatroom_list")

	filter := bson.D{{
		Key:   "chatroom_id",
		Value: models.GoUUIDToMongoUUID(chatroomID),
	}}

	var noSQLChatroomEntry models.NoSQLChatroomEntry
	var chatroomEntry models.ChatroomEntry
	if err := collection.FindOne(ctx, filter).Decode(&noSQLChatroomEntry); err != nil {
		return chatroomEntry, errors.Wrapf(err, "failed to find chatroom with id: %s", chatroomID.String())
	}

	return noSQLChatroomEntry.ConvertToChatroomEntry(), nil
}

// @todo make parameters for input arguments
func (m MongoRepo) AddUserToChatroom(ctx context.Context, chatroomID uuid.UUID, userID uuid.UUID) error {
	collection := m.client.Database(database, nil).Collection("chatroom_users")

	insert := models.NoSQLChatroomUserEntry{
		ChatroomID: models.GoUUIDToMongoUUID(chatroomID),
		UserID:     models.GoUUIDToMongoUUID(userID),
	}
	if _, err := collection.InsertOne(ctx, insert); err != nil {
		return errors.Wrapf(err, "failed to insert user %s to chatroom %s", userID, chatroomID)
	}

	return nil
}

func (m MongoRepo) GetUserConnectedChatrooms(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	collection := m.client.Database(database, nil).Collection("chatroom_users")

	filter := bson.D{{
		Key:   "user_id",
		Value: models.GoUUIDToMongoUUID(userID),
	}}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find chatroom users")
	}

	var chatroomIDs []uuid.UUID
	for cursor.Next(ctx) {
		var result models.NoSQLChatroomUserEntry
		if err := cursor.Decode(&result); err != nil {
			return nil, errors.Wrap(err, "failed to decode chatroom users")
		}

		chatroomIDs = append(chatroomIDs, models.MongoUUIDToGoUUID(result.ChatroomID))
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read chatroom users from cursor")
	}

	return chatroomIDs, nil
}
