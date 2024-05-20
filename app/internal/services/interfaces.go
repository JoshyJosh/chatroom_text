package services

import (
	"chatroom_text/internal/models"
	"context"

	"github.com/google/uuid"
)

type UserServicer interface {
	// SendMessage takes an unmarshalled websocket message and appends a user ID and timestamp.
	SendMessage(ctx context.Context, msg models.WSTextMessage)
	// ReceiveMessage takes a marshalled websocket message and appends a user ID and timestamp.
	ReceiveMessage(msg models.WSTextMessageBytes)
	// RemoveUser removes user from chatroom and user roster.
	RemoveUser()
	// EnterChatroom adds user to chatroom and retrieves its logs.
	EnterChatroom(ctx context.Context, chatroomID uuid.UUID, addUser bool) error

	// CreateChatroom creates a chatroom and adds its creator.
	CreateChatroom(ctx context.Context, msg models.WSChatroomCreateMessage)
	// UpdateChatroom updates a chatroom.
	UpdateChatroom(ctx context.Context, msg models.WSChatroomUpdateMessage)
	// UpdateChatroom deletes a chatroom.
	DeleteChatroom(ctx context.Context, msg models.WSChatroomDeleteMessage)

	// InitialConnect handles initial user connection and adds them to main chat.
	InitialConnect(ctx context.Context) error

	// Handler for listening to incoming messages from user related chatrooms.
	ListenForMessages(ctx context.Context)
}
