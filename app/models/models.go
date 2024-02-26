package models

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WSMessage struct {
	Text      string    `json:"msg"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userID"`
	UserName  string    `json:"userName"`
}

type AuthUserData struct {
	ID   uuid.UUID
	Name string
}

type User struct {
	ID        uuid.UUID
	Name      string
	WriteChan chan []byte
}

type ChatroomLog struct {
	ChatroomID uuid.UUID
	Timestamp  time.Time
	Text       string
	UserID     uuid.UUID
	UserName   string
}

type ChatroomLogMongo struct {
	ChatroomID primitive.Binary `bson:"chatroom_id"`
	Timestamp  time.Time        `bson:"timestamp"`
	Text       string           `bson:"text"`
	UserID     primitive.Binary `bson:"user_id"`
	UserName   string           `bson:"user_name"`
}

type SelectDBMessagesParams struct {
	TimestampFrom time.Time
	ChatroomID    uuid.UUID
}

type InsertDBMessagesParams struct {
	ChatroomID uuid.UUID
	Timestamp  time.Time
	UserID     uuid.UUID
	UserName   string
	Text       string
}

var MainChatUUID uuid.UUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// StandardizeTime rounds time.Time to milliseconds due to mongos Date type.
func StandardizeTime(t time.Time) time.Time {
	return t.Round(time.Millisecond)
}

func (clm ChatroomLogMongo) ConvertToChatroomLog() ChatroomLog {
	return ChatroomLog{
		ChatroomID: MongoUUIDToGoUUID(clm.ChatroomID),
		UserID:     MongoUUIDToGoUUID(clm.UserID),
		UserName:   clm.UserName,
		Timestamp:  clm.Timestamp,
		Text:       clm.Text,
	}
}

func (params InsertDBMessagesParams) ConvertToChatroomLogMongo() ChatroomLogMongo {
	return ChatroomLogMongo{
		ChatroomID: primitive.Binary{Subtype: 0x04, Data: []byte(params.ChatroomID[:])},
		UserID:     primitive.Binary{Subtype: 0x04, Data: []byte(params.UserID[:])},
		UserName:   params.UserName,
		Text:       params.Text,
		Timestamp:  params.Timestamp,
	}
}

func MongoUUIDToGoUUID(pUUID primitive.Binary) uuid.UUID {
	return uuid.UUID(pUUID.Data[:])
}

func GoUUIDToMongoUUID(gUUID uuid.UUID) primitive.Binary {
	return primitive.Binary{Subtype: 0x04, Data: []byte(gUUID[:])}
}
