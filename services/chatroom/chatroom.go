package chatroom

import (
	"chatroom_text/repo"
	chatroomws "chatroom_text/repo/chatroomws"
	"chatroom_text/services"
)

type Chatroom struct {
	repo repo.ChatroomRepoer
}

func GetChatroomServicer() services.ChatroomServicer {
	return Chatroom{
		repo: chatroomws.GetChatroom(),
	}
}

func (c Chatroom) AddNewUser(user services.UserServicer) string {
	return c.repo.AddNewUser(user)
}

func (c Chatroom) AddUser(id string, user services.UserServicer) error {
	return c.repo.AddUser(id, user)
}
