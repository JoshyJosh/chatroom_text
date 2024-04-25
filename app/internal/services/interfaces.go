package services

import (
	"chatroom_text/internal/models"
	"context"

	"github.com/google/uuid"
)

type UserServicer interface {
	// ReadMessage takes an unmarshalled websocket message and appends a user ID and timestamp.
	ReadMessage(ctx context.Context, msg models.WSTextMessage)
	// ReadMessage takes a marshalled websocket message and appends a user ID and timestamp.
	WriteMessage(msgRaw []byte)
	// RemoveUser removes user from chatroom and user roster.
	RemoveUser()
	// Add user to chatroom and retrieve its logs.
	EnterChatroom(ctx context.Context, chatroomID uuid.UUID, addUser bool) error

	CreateChatroom(ctx context.Context, msg models.WSChatroomCreateMessage)
	UpdateChatroom(ctx context.Context, msg models.WSChatroomUpdateMessage)
	DeleteChatroom(ctx context.Context, msg models.WSChatroomDeleteMessage)

	InitialConnect(ctx context.Context) error
}
