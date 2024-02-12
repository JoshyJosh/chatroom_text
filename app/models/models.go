package models

import (
	"time"

	"github.com/google/uuid"
)

type WSMessage struct {
	Text      string    `json:"msg"`
	Timestamp time.Time `json:"timestamp"`
	ClientID  string    `json:"clientID"` // should corelate with WSClient ID
}

type User struct {
	ID        uuid.UUID
	WriteChan chan []byte
}

type ChatroomLog struct {
	ChatroomID uuid.UUID `bson:"chatroom_id"`
	Timestamp  time.Time `bson:"timestamp"`
	Text       string    `bson:"text"`
	ClientID   uuid.UUID `bson:"client_id"`
}

type GetDBMessagesParams struct {
	TimestampFrom time.Time
	ChatroomID    uuid.UUID
}

type SetDBMessagesParams struct {
	ChatroomID uuid.UUID
	Timestamp  time.Time
	ClientID   uuid.UUID
	Text       string
}

var MainChatUUID uuid.UUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
