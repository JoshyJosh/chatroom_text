package nosql

import (
	"chatroom_text/internal/models"
	"chatroom_text/internal/repo"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepo struct {
	client *mongo.Client
	logger *log.Entry
}

var uri, database string

func InitAddr() {
	uri = os.Getenv("MONGODB_URI")
	if uri == "" {
		panic("missing MONGODB_URI env variable")
	}

	database = os.Getenv("MONGODB_DB")
	if uri == "" {
		panic("missing MONGODB_DB env variable")
	}
}

func GetChatroomNoSQLRepoer(ctx context.Context) (repo.ChatroomLogger, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongo instance")
	}

	logger := log.WithField("stage", "nosqlRepoer")

	return MongoRepo{
		client: client,
		logger: logger,
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
	if err != nil && !strings.Contains(err.Error(), "no responses remaining") {
		return nil, errors.Wrap(err, "failed to get cursor chatroom logs")
	}
	defer cursor.Close(ctx)

	var results []models.ChatroomLog
	for cursor.Next(ctx) {
		var result models.NoSQLChatroomLog
		if err := cursor.Decode(&result); err != nil {
			m.logger.Error("failed to decode NoSQLChatroomUserEntry: ", cursor.Current.String())
			continue
		}

		results = append(results, result.ConvertToChatroomLog())
	}

	if err := cursor.Err(); err != nil {
		// For mtest EOF errors.
		if !strings.Contains(err.Error(), "no responses remaining") {
			return nil, errors.Wrap(err, "failed to read logs from cursor")
		}
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

func (m MongoRepo) CreateChatroom(ctx context.Context, params models.CreateChatroomParams) (uuid.UUID, error) {
	chatroomUUID := uuid.New()

	insertDoc := models.NoSQLChatroomEntry{
		Name:       params.ChatroomName,
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
					return chatroomUUID, errors.Wrapf(err, "failed to create chatroom, name \"%s\" already taken", params.ChatroomName)
				}

				// Retry with different uuid.
				if strings.Contains(err.Error(), "chatroom_id") {
					chatroomUUID = uuid.New()
					continue
				}
			}

			// For mtest EOF errors.
			if strings.Contains(err.Error(), "no responses remaining") {
				break
			}

			return chatroomUUID, errors.Wrap(err, "failed to create chatroom")
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
func (m MongoRepo) SelectChatroomEntry(ctx context.Context, chatroomID uuid.UUID) (models.ChatroomEntry, error) {
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

	var res bson.M
	err := collection.FindOne(
		ctx,
		bson.D{
			{
				Key:   "chatroom_id",
				Value: models.GoUUIDToMongoUUID(chatroomID),
			},
			{
				Key:   "user_id",
				Value: models.GoUUIDToMongoUUID(userID),
			},
		},
	).Decode(&res)

	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return errors.Wrap(err, "failed to find if user is connected to chatroom")
	}

	if res != nil {
		return nil
	}

	insert := models.NoSQLChatroomUserEntry{
		ChatroomID: models.GoUUIDToMongoUUID(chatroomID),
		UserID:     models.GoUUIDToMongoUUID(userID),
	}
	if _, err := collection.InsertOne(ctx, insert); err != nil {
		return errors.Wrapf(err, "failed to insert user %s to chatroom %s", userID, chatroomID)
	}

	return nil
}

func (m MongoRepo) SelectUserConnectedChatrooms(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
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

func (m MongoRepo) SelectChatroomUsers(ctx context.Context, chatroomID uuid.UUID) ([]models.User, error) {
	collection := m.client.Database(database, nil).Collection("chatroom_users")

	filter := mongo.Pipeline{
		{{Key: "$match",
			Value: bson.D{{Key: "chatroom_id", Value: models.GoUUIDToMongoUUID(chatroomID)}},
		}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users_list"},
			{Key: "localField", Value: "user_id"},
			{Key: "foreignField", Value: "user_id"},
			{Key: "as", Value: "user_data"},
		}}},
	}
	cursor, err := collection.Aggregate(
		ctx,
		filter,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find chatroom users")
	}

	var users []models.User
	for cursor.Next(ctx) {
		var result models.NoSQLChatroomUserEntry
		if err := cursor.Decode(&result); err != nil {
			return nil, errors.Wrap(err, "failed to decode chatroom users")
		}

		user := models.User{
			ID:   models.MongoUUIDToGoUUID(result.UserID),
			Name: result.NameData[0].UserName,
		}
		users = append(users, user)
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read chatroom users from cursor")
	}

	return users, nil
}

func (m MongoRepo) StoreUsername(ctx context.Context, user models.User) error {
	m.logger.Info("storing user")
	defer m.logger.Info("stored user")
	collection := m.client.Database(database, nil).Collection("users_list")

	// Convert user ID once and reuse it.
	mongoUserID := models.GoUUIDToMongoUUID(user.ID)
	var result bson.M
	err := collection.FindOne(
		ctx,
		bson.D{
			{Key: "user_id", Value: mongoUserID},
		},
	).Decode(&result)

	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		if errors.Is(err, mongo.ErrNoDocuments) {
			collection.InsertOne(
				ctx,
				bson.D{
					{Key: "user_id", Value: mongoUserID},
					{Key: "user_name", Value: user.Name},
				},
			)

			return nil
		}

		m.logger.Error(err)
	}

	return err
}
