package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type WSMessage struct {
	Text      string    `json:"msg"`
	Timestamp time.Time `json:"timestamp"`
	ClientID  string    `json:"clientID"` // should corelate with WSClient ID
}

type User struct {
	ID        string
	WriteChan chan []byte
}

type ChatroomLog struct {
	ChatroomID sql.NullString `db:"chatroom_id"`
	Timestamp  time.Time      `db:"log_timestamp"`
	Text       string         `db:"log_text"`
	ClientID   sql.NullString `db:"client_id"`
}

type GetDBMessagesParams struct {
	TimestampFrom time.Time
	ChatroomID    uuid.UUID
}

type SetDBMessagesParams struct {
	ChatroomID uuid.UUID
	Timestamp  time.Time
	ClientID   string
	Text       string
}

var MainChatUUID uuid.UUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
