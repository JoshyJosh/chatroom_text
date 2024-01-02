package repo

import (
	"chatroom_text/models"
)

type ChatroomRepoer interface {
	// AddUser adds an existing user to chatroom, returns error if user exists.
	AddUser(user models.User) error
	// RemoveUser removes user from chatroom.
	RemoveUser(id string)
	// ReceiveMessage receives message from user to chatroom.
	ReceiveMessage(msg models.WSMessage)
	// DistributeMessage Distributes messages to all users in chatroom.
	DistributeMessage(message []byte)
}

type UserRepoer interface {
	// AddUser adds an existing user to chatroom, returns error if user exists.
	AddUser(models.User) string
	// ReceiveMessage receives message from user to chatroom.
	RemoveID(id string)
}

type ChatroomLogger interface {
	GetChatroomLogs(params models.GetDBMessagesParams) ([]models.ChatroomLog, error)
	SetChatroomLogs(params models.SetDBMessagesParams) error
}
