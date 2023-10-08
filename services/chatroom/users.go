package chatroom

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	chatroomws "chatroom_text/repo/chatroomws"
	services "chatroom_text/services"
	"time"
)

type User struct {
	user         models.UserWS
	userRepo     repo.UserRepoer
	chatroomRepo repo.ChatroomRepoer
}

func GetUserServicer(writeChan chan []byte) (services.UserServicer, error) {
	userService := User{
		chatroomRepo: chatroomws.GetChatroomRepoer(),
		userRepo:     chatroomws.GetUserRepoer(),
	}

	userService.user = models.UserWS{
		WriteChan: writeChan,
	}

	userService.user.ID = userService.userRepo.AddUser(userService.user)
	if err := userService.chatroomRepo.AddUser(userService.user); err != nil {
		return nil, err
	}

	return userService, nil
}

func (u User) ReadMessage(msg models.WSMessage) {
	msg.Timestamp = time.Now()
	msg.ClientID = u.user.ID
	u.chatroomRepo.ReceiveMessage(msg)
}

func (u User) WriteMessage(msgRaw []byte) {
	u.user.WriteChan <- msgRaw
}

func (u User) RemoveUser() {
	u.chatroomRepo.RemoveUser(u.user.ID)
	u.userRepo.RemoveID(u.user.ID)
}
