package repo

import (
	"chatroom_text/models"
	"chatroom_text/services"
)

type ChatroomRepoer interface {
	// AddNewUser adds a user to a chatroom and return a new genrated ID.
	AddNewUser(user services.UserServicer) (id string)
	// AddUser adds an existing user to chatroom, returns error if user exists.
	AddUser(id string, user services.UserServicer) error
	// ReceiveMessage receives message from user to chatroom.
	ReceiveMessage(msg models.WSMessage)
	// DistributeMessage Distributes messages to all users in chatroom.
	DistributeMessage(message []byte)
}
