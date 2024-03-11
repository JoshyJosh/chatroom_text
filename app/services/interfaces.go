package services

import (
	"chatroom_text/models"
	"context"
)

type UserServicer interface {
	// ReadMessage takes an unmarshalled websocket message and appends a user ID and timestamp.
	ReadMessage(ctx context.Context, msg models.WSTextMessage)
	// ReadMessage takes a marshalled websocket message and appends a user ID and timestamp.
	WriteMessage(msgRaw []byte)
	// RemoveUser removes user from chatroom and user roster.
	RemoveUser()
	// Add user to chatroom and retrieve its logs.
	EnterChatroom(ctx context.Context) error
}

type ChatroomServicer interface {
	CreateChatroom(ctx context.Context, msg models.WSCreateChatroomMessage) models.WSCreateChatroomConfirmationMessage
	UpdateChatroom(ctx context.Context, msg models.WSUpdateChatroomMessage) models.WSUpdateChatroomConfirmationMessage
	DeleteChatroom(ctx context.Context, msg models.WSDeleteChatroomMessage) models.WSDeleteChatroomConfirmationMessage
}
