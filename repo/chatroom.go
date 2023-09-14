package repo

import "chatroom_text/models"

type Chatroomer interface {
	AddUser(Userer) error
	RemoveUser(Userer) error
	ListUsers() []string
	AddMessage(models.Message) error
}
