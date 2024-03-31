package models

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WSMessage struct {
	TextMessage     *WSTextMessage   `json:"text,omitempty"`
	ChatroomMessage *ChatroomMessage `json:"chatroom,omitempty"`
}

type WSTextMessage struct {
	Text       string    `json:"msg"`
	Timestamp  time.Time `json:"timestamp"`
	UserID     string    `json:"userID"`
	UserName   string    `json:"userName"`
	ChatroomID string    `json:"chatroomID"`
}

type ChatroomMessage struct {
	Create *WSChatroomCreateMessage `json:"create,omitempty"`
	Update *WSChatroomUpdateMessage `json:"update,omitempty"`
	Delete *WSChatroomDeleteMessage `json:"delete,omitempty"`

	// Used for users to invite users in chatrooms.
	Enter *WSChatroomEnterMessage `json:"enter,omitempty"`
}

type WSChatroomCreateMessage struct {
	ChatroomName string   `json:"chatroomName"`
	InviteUsers  []string `json:"inviteUsers"`
}

type WSChatroomUpdateMessage struct {
	ChatroomID      string   `json:"chatroomID"`
	NewChatroomName string   `json:"newChatroomName"`
	InviteUsers     []string `json:"inviteUsers"`
	RemoveUsers     []string `json:"removeUsers"`
}

type WSChatroomDeleteMessage struct {
	ChatroomID string `json:"chatroomID"`
}

type WSChatroomEnterMessage struct {
	ChatroomName string `json:"chatroomName"`
	ChatroomID   string `json:"chatroomID"`
}

type WSCreateChatroomConfirmationMessage struct {
	Success bool `json:"success"`
}

type WSUpdateChatroomConfirmationMessage struct {
	Success bool `json:"success"`
}

type WSDeleteChatroomConfirmationMessage struct {
	Success bool `json:"success"`
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

type ChatroomEntry struct {
	ChatroomID uuid.UUID
	Name       string
	IsActive   bool
}

type NoSQLChatroomLog struct {
	ChatroomID primitive.Binary `bson:"chatroom_id"`
	Timestamp  time.Time        `bson:"timestamp"`
	Text       string           `bson:"text"`
	UserID     primitive.Binary `bson:"user_id"`
	UserName   string           `bson:"user_name"`
}

type NoSQLChatroomEntry struct {
	Name       string           `bson:"name"`
	ChatroomID primitive.Binary `bson:"chatroom_id"`
	IsActive   bool             `bson:"is_active"`
}

type NoSQLChatroomUserEntry struct {
	ChatroomID primitive.Binary `bson:"chatroom_id"`
	UserID     primitive.Binary `bson:"user_id"`
}

// @todo actuall prepare use for TimestampFrom
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

const MainChatName = "mainChat"

// StandardizeTime rounds time.Time to milliseconds due to mongos Date type.
func StandardizeTime(t time.Time) time.Time {
	return t.Round(time.Millisecond)
}

func (nscl NoSQLChatroomLog) ConvertToChatroomLog() ChatroomLog {
	return ChatroomLog{
		ChatroomID: MongoUUIDToGoUUID(nscl.ChatroomID),
		UserID:     MongoUUIDToGoUUID(nscl.UserID),
		UserName:   nscl.UserName,
		Timestamp:  nscl.Timestamp,
		Text:       nscl.Text,
	}
}

func (params InsertDBMessagesParams) ConvertToNoSQLChatroomLog() NoSQLChatroomLog {
	return NoSQLChatroomLog{
		ChatroomID: primitive.Binary{Subtype: 0x04, Data: []byte(params.ChatroomID[:])},
		UserID:     primitive.Binary{Subtype: 0x04, Data: []byte(params.UserID[:])},
		UserName:   params.UserName,
		Text:       params.Text,
		Timestamp:  params.Timestamp,
	}
}

func (nsce NoSQLChatroomEntry) ConvertToChatroomEntry() ChatroomEntry {
	return ChatroomEntry{
		ChatroomID: MongoUUIDToGoUUID(nsce.ChatroomID),
		Name:       nsce.Name,
		IsActive:   nsce.IsActive,
	}
}

func MongoUUIDToGoUUID(pUUID primitive.Binary) uuid.UUID {
	return uuid.UUID(pUUID.Data[:])
}

func GoUUIDToMongoUUID(gUUID uuid.UUID) primitive.Binary {
	return primitive.Binary{Subtype: 0x04, Data: []byte(gUUID[:])}
}
