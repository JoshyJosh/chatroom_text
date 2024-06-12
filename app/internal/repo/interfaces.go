package repo

import (
	"chatroom_text/internal/models"
	"context"

	"github.com/google/uuid"
)

type ChatroomMessageBroker interface {
	// AddUser adds an existing user to chatroom, returns error if user exists.
	AddUser(chatroomID uuid.UUID) error
	// RemoveUser removes user from chatroom.
	RemoveUser(chatroomID uuid.UUID) error
	// DistributeMessage Distributes messages to all users in chatroom.
	DistributeMessage(ctx context.Context, chatroomID uuid.UUID, msgBytes models.WSTextMessageBytes) error
	// Listen listens to incoming messages and passes them to user message channel.
	Listen(ctx context.Context, msgBytesChan chan<- models.WSTextMessageBytes)

	DistributeUserEntryMessage(ctx context.Context, chatroomID uuid.UUID, msgBytes models.WSUserEntry) error
}

type ChatroomLogger interface {
	// SelectChatroomLogs selects logs sorted by timestamp.
	SelectChatroomLogs(ctx context.Context, params models.SelectDBMessagesParams) ([]models.ChatroomLog, error)
	// SelectChatroomLogs stores chatroom logs.
	InsertChatroomLogs(ctx context.Context, params models.InsertDBMessagesParams) error

	// Create chatroom with initially invited user IDs.
	CreateChatroom(ctx context.Context, params models.CreateChatroomParams) (uuid.UUID, error)
	// Update chatroom with add remove user IDs.
	UpdateChatroom(ctx context.Context, chatroomID uuid.UUID, newName string, addUsers []string, removeUsers []string) error
	// Delete chatroom.
	DeleteChatroom(ctx context.Context, chatroomID uuid.UUID) error
	// GetChatroomEntry retrieves chatroom name, ID and active attribute of a chatroom.
	SelectChatroomEntry(ctx context.Context, chatroomID uuid.UUID) (models.ChatroomEntry, error)

	// AddUserToChatroom adds user to chatroom.
	AddUserToChatroom(ctx context.Context, chatroomID uuid.UUID, userID uuid.UUID) error
	//SelectChatroomUsers retrieves users that are currently in the chatroom.
	SelectChatroomUsers(ctx context.Context, chatroomID uuid.UUID) ([]models.User, error)
	// GetUserConnectedChatrooms retrieves list of chatrooms which contains this user.
	SelectUserConnectedChatrooms(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	// Store username to map uuid to user.
	StoreUsername(ctx context.Context, user models.User) error
}
