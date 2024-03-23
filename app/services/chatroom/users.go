package chatroom

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	chatroomRepo "chatroom_text/repo/mem"
	chatroomNoSQL "chatroom_text/repo/nosql"
	services "chatroom_text/services"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
)

type User struct {
	user                models.User
	userRepo            repo.UserRepoer
	chatroomRepos       *sync.Map // map which key is the chat uuid and value is repo.ChatroomRepoer
	chatroomNoSQLRepoer repo.ChatroomNoSQLRepoer
}

func GetUserServicer(ctx context.Context, writeChan chan []byte, userData models.AuthUserData) (services.UserServicer, error) {
	chatroomNoSQLRepoer, err := chatroomNoSQL.GetChatroomNoSQLRepoer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logger for user servicer")
	}

	userService := User{
		userRepo:            chatroomRepo.GetUserRepoer(),
		chatroomNoSQLRepoer: chatroomNoSQLRepoer,
		chatroomRepos:       &sync.Map{},
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

func (u User) EnterChatroom(ctx context.Context, chatroomName string) error {
	if chatroomName == "" {
		return errors.New("cannot enter empty chatroom name")
	}

	chatroomID, err := u.chatroomNoSQLRepoer.GetChatroomUUID(ctx, chatroomName)
	if err != nil {
		return err
	}

	slog.Info("chatroomID: ", chatroomID.String())
	enteredChatroom := chatroomRepo.GetChatroomRepoer(chatroomID)
	u.chatroomRepos.Store(chatroomID, enteredChatroom)
	if err := enteredChatroom.AddUser(u.user); err != nil {
		return err
	}

	msgRaw, err := json.Marshal(models.WSMessage{
		ChatroomMessage: &models.ChatroomMessage{
			Enter: &models.WSChatroomEnterMessage{
				ChatroomName: chatroomName,
				ChatroomID:   chatroomID.String(),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal chatroom entry message")
	}

	u.WriteMessage(msgRaw)

	logs, err := u.chatroomNoSQLRepoer.SelectChatroomLogs(ctx, models.SelectDBMessagesParams{
		ChatroomID: chatroomID,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get chatroom logs")
	}

	for _, l := range logs {
		msg := models.WSTextMessage{
			Text:       l.Text,
			Timestamp:  l.Timestamp,
			UserID:     l.UserID.String(),
			UserName:   l.UserName,
			ChatroomID: chatroomID.String(),
		}

		msgRaw, err := json.Marshal(models.WSMessage{
			TextMessage: &msg,
		})
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal chatroom logs")
		}

		u.WriteMessage(msgRaw)
	}

	// Send entry text message to all people in chatroom.
	u.ReadMessage(ctx, models.WSTextMessage{
		Text:       "entered chat",
		Timestamp:  models.StandardizeTime(time.Now()),
		UserID:     u.user.ID.String(),
		UserName:   u.user.Name,
		ChatroomID: chatroomID.String(),
	})

	return nil
}

func (u User) ReadMessage(ctx context.Context, msg models.WSTextMessage) {
	msg.Timestamp = models.StandardizeTime(time.Now())
	msg.UserID = u.user.ID.String()
	msg.UserName = u.user.Name

	chatroomID, err := uuid.Parse(msg.ChatroomID)
	if err != nil {
		slog.Error("failed to parse chatroomID: ", err.Error())
	}

	err = u.chatroomNoSQLRepoer.InsertChatroomLogs(ctx, models.InsertDBMessagesParams{
		ChatroomID: chatroomID,
		Timestamp:  msg.Timestamp,
		UserName:   u.user.Name,
		UserID:     u.user.ID,
		Text:       msg.Text,
	})
	if err != nil {
		// @todo propagate error message to websocket
		slog.Error("failed to insert chatroom log", err.Error())
	}

	chatroomRepo, loaded := u.chatroomRepos.Load(chatroomID)
	if !loaded {
		slog.Error("failed to load chatroom repo with id: ", msg.ChatroomID)
	}
	chatroomRepo.(repo.ChatroomRepoer).ReceiveMessage(msg)
}

func (u User) WriteMessage(msgRaw []byte) {
	u.user.WriteChan <- msgRaw
}

func (u User) RemoveUser() {
	u.chatroomRepos.Range(func(key, value any) bool {
		value.(repo.ChatroomRepoer).RemoveUser(u.user.ID)
		return true
	})
	u.userRepo.RemoveID(u.user.ID)
}

// @todo consider just writing the error in write channel
func (u User) CreateChatroom(ctx context.Context, msg models.WSChatroomCreateMessage) {
	slog.Info(fmt.Sprintf("creating chatroom: %s", msg.ChatroomName))
	if err := u.chatroomNoSQLRepoer.CreateChatroom(ctx, msg.ChatroomName, msg.InviteUsers); err != nil {
		slog.Error("failed to create chatroom ", err)
		return
	}

	if err := u.EnterChatroom(ctx, msg.ChatroomName); err != nil {
		slog.Error("failed to enter chatroom ", err)
		return
	}
}

// @todo implement adding and removing users.
func (u User) UpdateChatroom(ctx context.Context, msg models.WSChatroomUpdateMessage) {
	chatroomID, err := uuid.Parse(msg.ChatroomID)
	if err != nil {
		slog.Error("failed to parse uuid: ", err)
		return
	}

	if models.MainChatUUID == chatroomID {
		slog.Error("cannot update main chat")
		return
	}

	if msg.NewChatroomName == "" {
		slog.Error("cannot set empty name for chatroom")
	}

	if err := u.chatroomNoSQLRepoer.UpdateChatroom(ctx, chatroomID, msg.NewChatroomName, nil, nil); err != nil {
		slog.Error("failed to update chatroom: ", err)
		return
	}

	updateMessage, err := json.Marshal(models.WSMessage{
		ChatroomMessage: &models.ChatroomMessage{
			Update: &models.WSChatroomUpdateMessage{
				ChatroomID:      msg.ChatroomID,
				NewChatroomName: msg.NewChatroomName,
			},
		},
	})
	if err != nil {
		slog.Error("failed to marshal chatroom update message: ", err)
		return
	}

	chatroomRepo, ok := u.chatroomRepos.Load(chatroomID)
	if !ok {
		slog.Error("chatroom repo with chatroomID does not exist: %s", chatroomID.String())
		return
	}

	chatroomRepo.(repo.ChatroomRepoer).DistributeMessage(updateMessage)
}

func (u User) DeleteChatroom(ctx context.Context, msg models.WSChatroomDeleteMessage) {
	chatroomID, err := uuid.Parse(msg.ChatroomID)
	if err != nil {
		slog.Error("failed to parse uuid: ", err)
		return
	}

	if models.MainChatUUID == chatroomID {
		slog.Error("cannot delete main chat")
		return
	}

	chatroomRepo, ok := u.chatroomRepos.Load(chatroomID)
	if !ok {
		slog.Error("you are not part of the chatroom repo or it does not exist")
		return
	}

	if err := u.chatroomNoSQLRepoer.DeleteChatroom(ctx, chatroomID); err != nil {
		slog.Error("failed to delete chatroom: ", err)
		return
	}

	deleteMessage, err := json.Marshal(models.WSMessage{
		ChatroomMessage: &models.ChatroomMessage{
			Delete: &models.WSChatroomDeleteMessage{
				ChatroomID: msg.ChatroomID,
			},
		},
	})
	if err != nil {
		slog.Error("failed to marshal chatroom delete message: ", err)
		return
	}

	chatroomRepo.(repo.ChatroomRepoer).DistributeMessage(deleteMessage)
}
