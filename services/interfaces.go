package services

import (
	"chatroom_text/models"
)

type UserServicer interface {
	// ReadMessage takes an unmarshalled websocket message and appends a user ID and timestamp.
	ReadMessage(msg models.WSMessage)
	// ReadMessage takes a marshalled websocket message and appends a user ID and timestamp.
	WriteMessage(msgRaw []byte)
	// RemoveUser removes user from chatroom and user roster.
	RemoveUser()
	// Add user to chatroom and retrieve its logs.
	EnterChatroom() error
}
