package repo

type Userer interface {
	SendMessage(message string, chatroom Chatroomer) error
	ListChatrooms() []string
}
