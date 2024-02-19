package models

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WSMessage struct {
	Text      string    `json:"msg"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"clientID"`
}

type User struct {
	ID        uuid.UUID
	WriteChan chan []byte
}

type ChatroomLog struct {
	ChatroomID uuid.UUID
	Timestamp  time.Time
	Text       string
	UserID     uuid.UUID
}

type ChatroomLogMongo struct {
	ChatroomID primitive.Binary `bson:"chatroom_id"`
	Timestamp  time.Time        `bson:"timestamp"`
	Text       string           `bson:"text"`
	UserID     primitive.Binary `bson:"client_id"`
}

type SelectDBMessagesParams struct {
	TimestampFrom time.Time
	ChatroomID    uuid.UUID
}

type InsertDBMessagesParams struct {
	ChatroomID uuid.UUID
	Timestamp  time.Time
	UserID     uuid.UUID
	Text       string
}

var MainChatUUID uuid.UUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// StandardizeTime rounds time.Time to milliseconds due to mongos Date type.
func StandardizeTime(t time.Time) time.Time {
	return t.Round(time.Millisecond)
}
