package nosql

import (
	"chatroom_text/models"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestSelectChatroomLogs(t *testing.T) {
	ctx := context.Background()

	opts := mtest.NewOptions().ClientType(mtest.Mock)

	mt := mtest.New(t, opts)

	mt.Run("Success: Successfuly select chatroom logs", func(mt *mtest.T) {
		a := assert.New(t)

		// Time must be UTC in order for mtest client to work.
		now, err := time.ParseInLocation(time.DateTime, "2024-03-31 12:00:00", time.UTC)
		if err != nil {
			t.Fatal(err)
		}

		user1ID := uuid.New()
		user2ID := uuid.New()
		user3ID := uuid.New()

		cursorBatch := []bson.D{
			{
				{Key: "chatroom_id", Value: models.GoUUIDToMongoUUID(models.MainChatUUID)},
				{Key: "timestamp", Value: now.Add(-30 * time.Minute)},
				{Key: "text", Value: "test text 1"},
				{Key: "user_id", Value: user1ID},
				{Key: "user_name", Value: "testuser1"},
			},
			{
				{Key: "chatroom_id", Value: models.GoUUIDToMongoUUID(models.MainChatUUID)},
				{Key: "timestamp", Value: now.Add(-25 * time.Minute)},
				{Key: "text", Value: "test text 2"},
				{Key: "user_id", Value: user2ID},
				{Key: "user_name", Value: "testuser2"},
			},
			{
				{Key: "chatroom_id", Value: models.GoUUIDToMongoUUID(models.MainChatUUID)},
				{Key: "timestamp", Value: now},
				{Key: "text", Value: "test text 3"},
				{Key: "user_id", Value: user3ID},
				{Key: "user_name", Value: "testuser3"},
			},
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"chatroom.chat_logs",
			mtest.FirstBatch,
			cursorBatch...,
		))

		mockMongoRepo := MongoRepo{
			client: mt.Client,
		}

		logs, err := mockMongoRepo.SelectChatroomLogs(ctx, models.SelectDBMessagesParams{
			TimestampFrom: time.Now(),
			ChatroomID:    models.MainChatUUID,
		})

		a.Nil(err)
		a.Len(logs, 3)

		a.Equal(logs[0], models.ChatroomLog{
			ChatroomID: models.MainChatUUID,
			Timestamp:  now.Add(-30 * time.Minute),
			Text:       "test text 1",
			UserID:     user1ID,
			UserName:   "testuser1",
		})

		a.Equal(logs[1], models.ChatroomLog{
			ChatroomID: models.MainChatUUID,
			Timestamp:  now.Add(-25 * time.Minute),
			Text:       "test text 2",
			UserID:     user2ID,
			UserName:   "testuser2",
		})

		a.Equal(logs[2], models.ChatroomLog{
			ChatroomID: models.MainChatUUID,
			Timestamp:  now,
			Text:       "test text 3",
			UserID:     user3ID,
			UserName:   "testuser3",
		})
	})

	mt.Run("Success: Successfuly select chatroom logs with undecodable documents", func(mt *mtest.T) {
		a := assert.New(t)

		// Time must be UTC in order for mtest client to work.
		now, err := time.ParseInLocation(time.DateTime, "2024-03-31 12:00:00", time.UTC)
		if err != nil {
			t.Fatal(err)
		}

		user2ID := uuid.New()
		user3ID := uuid.New()

		cursorBatch := []bson.D{
			{
				{Key: "chatroom_id", Value: models.GoUUIDToMongoUUID(models.MainChatUUID)},
				{Key: "timestamp", Value: now.Add(-30 * time.Minute)},
				{Key: "text", Value: "test text 1"},
				{Key: "user_id", Value: "not an id"},
				{Key: "user_name", Value: "testuser1"},
			},
			{
				{Key: "chatroom_id", Value: models.GoUUIDToMongoUUID(models.MainChatUUID)},
				{Key: "timestamp", Value: now.Add(-25 * time.Minute)},
				{Key: "text", Value: "test text 2"},
				{Key: "user_id", Value: user2ID},
				{Key: "user_name", Value: "testuser2"},
			},
			{
				{Key: "chatroom_id", Value: models.GoUUIDToMongoUUID(models.MainChatUUID)},
				{Key: "timestamp", Value: now},
				{Key: "text", Value: "test text 3"},
				{Key: "user_id", Value: user3ID},
				{Key: "user_name", Value: "testuser3"},
			},
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"chatroom.chat_logs",
			mtest.FirstBatch,
			cursorBatch...,
		))

		mockMongoRepo := MongoRepo{
			client: mt.Client,
		}

		logs, err := mockMongoRepo.SelectChatroomLogs(ctx, models.SelectDBMessagesParams{
			TimestampFrom: time.Now(),
			ChatroomID:    models.MainChatUUID,
		})

		a.Nil(err)
		a.Len(logs, 2)

		a.Equal(logs[0], models.ChatroomLog{
			ChatroomID: models.MainChatUUID,
			Timestamp:  now.Add(-25 * time.Minute),
			Text:       "test text 2",
			UserID:     user2ID,
			UserName:   "testuser2",
		})

		a.Equal(logs[1], models.ChatroomLog{
			ChatroomID: models.MainChatUUID,
			Timestamp:  now,
			Text:       "test text 3",
			UserID:     user3ID,
			UserName:   "testuser3",
		})
	})

	mt.Run("Success: Empty select response", func(mt *mtest.T) {
		a := assert.New(t)
		mt.AddMockResponses()

		mockMongoRepo := MongoRepo{
			client: mt.Client,
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"chatroom.chat_logs",
			mtest.FirstBatch,
		))

		logs, err := mockMongoRepo.SelectChatroomLogs(ctx, models.SelectDBMessagesParams{
			TimestampFrom: time.Now(),
			ChatroomID:    models.MainChatUUID,
		})

		a.Nil(err)
		a.Len(logs, 0)
	})

	mt.Run("Failure: Error retrieved from cursor", func(mt *mtest.T) {
		a := assert.New(t)
		mockMongoRepo := MongoRepo{
			client: mt.Client,
		}

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    1,
			Message: mongo.ErrNoDocuments.Error(),
			Name:    "test error",
			Labels:  []string{},
		}))

		_, err := mockMongoRepo.SelectChatroomLogs(ctx, models.SelectDBMessagesParams{
			TimestampFrom: time.Now(),
			ChatroomID:    models.MainChatUUID,
		})

		a.Error(err)
	})
}
