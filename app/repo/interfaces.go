package repo

import (
	"chatroom_text/models"
	"context"

	"github.com/google/uuid"
)

type ChatroomRepoer interface {
	// AddUser adds an existing user to chatroom, returns error if user exists.
	AddUser(user models.User) error
	// RemoveUser removes user from chatroom.
	RemoveUser(id uuid.UUID)
	// ReceiveMessage receives message from user to chatroom.
	ReceiveMessage(msg models.WSMessage)
	// DistributeMessage Distributes messages to all users in chatroom.
	DistributeMessage(message []byte)
}

type UserRepoer interface {
	// AddUser adds an existing user to chatroom, returns error if user exists.
	AddUser(user models.User) uuid.UUID
	// ReceiveMessage receives message from user to chatroom.
	RemoveID(id uuid.UUID)
}

type ChatroomLogRepoer interface {
	GetChatroomLogs(ctx context.Context, params models.GetDBMessagesParams) ([]models.ChatroomLog, error)
	SetChatroomLogs(ctx context.Context, params models.SetDBMessagesParams) error
}
