package chatroom

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	chatroomRepo "chatroom_text/repo/mem"
	chatroomLogger "chatroom_text/repo/nosql"
	services "chatroom_text/services"
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
)

type User struct {
	user           models.User
	userRepo       repo.UserRepoer
	chatroomRepo   repo.ChatroomRepoer
	chatroomLogger repo.ChatroomLogRepoer
}

func GetUserServicer(ctx context.Context, writeChan chan []byte, userData models.AuthUserData) (services.UserServicer, error) {
	chatroomLogger, err := chatroomLogger.GetChatroomLogRepoer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logger for user servicer")
	}

	userService := User{
		userRepo:       chatroomRepo.GetUserRepoer(),
		chatroomRepo:   chatroomRepo.GetChatroomRepoer(),
		chatroomLogger: chatroomLogger,
	}

	userService.user = models.User{
		WriteChan: writeChan,
		Name:      userData.Name,
		ID:        userData.ID,
	}

	if err := userService.userRepo.AddUser(userService.user); err != nil {
		return nil, err
	}

	return userService, nil
}

func (u User) EnterChatroom(ctx context.Context) error {
	if err := u.chatroomRepo.AddUser(u.user); err != nil {
		return err
	}

	logs, err := u.chatroomLogger.SelectChatroomLogs(ctx, models.SelectDBMessagesParams{
		ChatroomID: models.MainChatUUID,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get chatroom logs")
	}

	for _, l := range logs {
		msg := models.WSTextMessage{
			Text:      l.Text,
			Timestamp: l.Timestamp,
			UserID:    l.UserID.String(),
			UserName:  l.UserName,
		}

		msgRaw, err := json.Marshal(msg)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal chatroom logs")
		}

		u.WriteMessage(msgRaw)
	}

	return nil
}

func (u User) ReadMessage(ctx context.Context, msg models.WSTextMessage) {
	msg.Timestamp = models.StandardizeTime(time.Now())
	msg.UserID = u.user.ID.String()
	msg.UserName = u.user.Name

	err := u.chatroomLogger.InsertChatroomLogs(ctx, models.InsertDBMessagesParams{
		ChatroomID: models.MainChatUUID,
		Timestamp:  msg.Timestamp,
		UserName:   u.user.Name,
		UserID:     u.user.ID,
		Text:       msg.Text,
	})
	if err != nil {
		// @todo propagate error message to websocket
		slog.Error("failed to insert chatroom log", err.Error())
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
