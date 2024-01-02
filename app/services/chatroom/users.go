package chatroom

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	chatroomLogger "chatroom_text/repo/db"
	chatroomRepo "chatroom_text/repo/mem"
	services "chatroom_text/services"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
)

type User struct {
	user           models.User
	userRepo       repo.UserRepoer
	chatroomRepo   repo.ChatroomRepoer
	chatroomLogger repo.ChatroomLogger
}

func GetUserServicer(writeChan chan []byte) (services.UserServicer, error) {
	userService := User{
		userRepo:       chatroomRepo.GetUserRepoer(),
		chatroomRepo:   chatroomRepo.GetChatroomRepoer(),
		chatroomLogger: chatroomLogger.GetChatroomLogger(),
	}

	userService.user = models.User{
		WriteChan: writeChan,
	}

	userService.user.ID = userService.userRepo.AddUser(userService.user)

	return userService, nil
}

func (u User) EnterChatroom() error {
	if err := u.chatroomRepo.AddUser(u.user); err != nil {
		return err
	}

	logs, err := u.chatroomLogger.GetChatroomLogs(models.GetDBMessagesParams{
		ChatroomID: models.MainChatUUID,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get chatroom logs")
	}

	for _, l := range logs {
		msg := models.WSMessage{
			Text:      l.Text,
			Timestamp: l.Timestamp,
			ClientID:  l.ClientID.String,
		}

		msgRaw, err := json.Marshal(msg)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal chatroom logs")
		}

		u.WriteMessage(msgRaw)
	}

	return nil
}

func (u User) ReadMessage(msg models.WSMessage) {
	msg.Timestamp = time.Now()
	msg.ClientID = u.user.ID

	err := u.chatroomLogger.SetChatroomLogs(models.SetDBMessagesParams{
		ChatroomID: models.MainChatUUID,
		Timestamp:  msg.Timestamp,
		ClientID:   msg.ClientID,
		Text:       msg.Text,
	})
	if err != nil {
		// @todo return error response
		slog.Error(err.Error())
	}
	u.chatroomRepo.ReceiveMessage(msg)
}

func (u User) WriteMessage(msgRaw []byte) {
	u.user.WriteChan <- msgRaw
}

func (u User) RemoveUser() {
	u.chatroomRepo.RemoveUser(u.user.ID)
	u.userRepo.RemoveID(u.user.ID)
}
