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
	ReceiveMessage(msg models.WSTextMessage)
	// DistributeMessage Distributes messages to all users in chatroom.
	DistributeMessage(message []byte)
}

type UserRepoer interface {
	// AddUser adds an existing user to chatroom, returns error if user exists.
	AddUser(user models.User) error
	// ReceiveMessage receives message from user to chatroom.
	RemoveID(id uuid.UUID)
}

type ChatroomNoSQLRepoer interface {
	SelectChatroomLogs(ctx context.Context, params models.SelectDBMessagesParams) ([]models.ChatroomLog, error)
	InsertChatroomLogs(ctx context.Context, params models.InsertDBMessagesParams) error
	// Create chatroom with initially invited user IDs.
	CreateChatroom(ctx context.Context, params models.CreateChatroomParams) (uuid.UUID, error)
	// Update chatroom with add remove user IDs.
	UpdateChatroom(ctx context.Context, chatroomID uuid.UUID, newName string, addUsers []string, removeUsers []string) error
	// Delete chatroom.
	DeleteChatroom(ctx context.Context, chatroomID uuid.UUID) error
	GetChatroomEntry(ctx context.Context, chatroomID uuid.UUID) (models.ChatroomEntry, error)
	AddUserToChatroom(ctx context.Context, chatroomID uuid.UUID, userID uuid.UUID) error
	GetUserConnectedChatrooms(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
