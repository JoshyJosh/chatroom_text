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

func GetUserServicer(ctx context.Context, writeChan chan []byte) (services.UserServicer, error) {
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
	}

	userService.user.ID = userService.userRepo.AddUser(userService.user)

	return userService, nil
}

func (u User) EnterChatroom(ctx context.Context) error {
	if err := u.chatroomRepo.AddUser(u.user); err != nil {
		return err
	}

	logs, err := u.chatroomLogger.SelectChatroomLogs(ctx, models.GetDBMessagesParams{
		ChatroomID: models.MainChatUUID,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get chatroom logs")
	}

	for _, l := range logs {
		msg := models.WSMessage{
			Text:      l.Text,
			Timestamp: l.Timestamp,
			ClientID:  l.ClientID.String(),
		}

		msgRaw, err := json.Marshal(msg)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal chatroom logs")
		}

		u.WriteMessage(msgRaw)
	}

	return nil
}

func (u User) ReadMessage(ctx context.Context, msg models.WSMessage) {
	msg.Timestamp = models.StandardizeTime(time.Now())
	msg.ClientID = u.user.ID.String()

	err := u.chatroomLogger.InsertChatroomLogs(ctx, models.SetDBMessagesParams{
		ChatroomID: models.MainChatUUID,
		Timestamp:  msg.Timestamp,
		ClientID:   u.user.ID,
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
