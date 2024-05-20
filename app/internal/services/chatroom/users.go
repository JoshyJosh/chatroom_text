package chatroom

import (
	"chatroom_text/internal/models"
	"chatroom_text/internal/repo"
	chatroomNoSQL "chatroom_text/internal/repo/nosql"
	"chatroom_text/internal/repo/rabbitmq"
	services "chatroom_text/internal/services"
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
	messageBroker       repo.ChatroomMessageBroker
	chatroomNoSQLRepoer repo.ChatroomLogger
}

func GetUserServicer(ctx context.Context, writeChan chan []byte, userData models.AuthUserData) (services.UserServicer, error) {
	chatroomNoSQLRepoer, err := chatroomNoSQL.GetChatroomNoSQLRepoer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logger for user servicer")
	}

	user := models.User{
		WriteChan: writeChan,
		Name:      userData.Name,
		ID:        userData.ID,
	}

	// @todo redefine to make separate entrances... Makes sense
	chatroomMessageBroker, err := rabbitmq.GetChatroomMessageBroker(user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chatroom message broker")
	}

	userService := User{
		messageBroker:       chatroomMessageBroker,
		chatroomNoSQLRepoer: chatroomNoSQLRepoer,
		user:                user,
	}

	return userService, nil
}

func (u User) EnterChatroom(ctx context.Context, chatroomID uuid.UUID, addUser bool) error {
	slog.Info("entering chatroomID: ", chatroomID.String())

	if err := u.messageBroker.AddUser(chatroomID); err != nil {
		return err
	}

	chatroomEntry, err := u.chatroomNoSQLRepoer.GetChatroomEntry(ctx, chatroomID)
	if err != nil {
		return errors.Wrap(err, "failed to get chatroom entry")
	}

	// @todo determine if users have actually been added to nosql entry
	// If user connects the first time mainChat adds the user if it is not listed
	if addUser {
		u.chatroomNoSQLRepoer.AddUserToChatroom(ctx, chatroomID, u.user.ID)
	}

	msgRaw, err := json.Marshal(models.WSMessage{
		ChatroomMessage: &models.ChatroomMessage{
			Enter: &models.WSChatroomEnterMessage{
				ChatroomName: chatroomEntry.Name,
				ChatroomID:   chatroomEntry.ChatroomID.String(),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal chatroom entry message")
	}

	u.ReceiveMessage(msgRaw)

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

		u.ReceiveMessage(msgRaw)
	}

	// Send entry text message to all people in chatroom.
	u.SendMessage(ctx, models.WSTextMessage{
		Text:       "entered chat",
		Timestamp:  models.StandardizeTime(time.Now()),
		UserID:     u.user.ID.String(),
		UserName:   u.user.Name,
		ChatroomID: chatroomID.String(),
	})

	return nil
}

func (u User) SendMessage(ctx context.Context, msg models.WSTextMessage) {
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

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		// @todo propagate error message to websocket
		slog.Error("failed to marshal chatroom log", err.Error())
	}

	u.messageBroker.DistributeMessage(ctx, msgBytes)
}

// Used to writing to this user directly, currently for initial noSQL data retrieval.
func (u User) ReceiveMessage(msgRaw models.WSTextMessageBytes) {
	u.user.WriteChan <- msgRaw
}

func (u User) ListenForMessages(ctx context.Context) {
	msgBytesChan := make(chan models.WSTextMessageBytes)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		u.messageBroker.Listen(msgBytesChan)
	}()
	for msgRaw := range msgBytesChan {
		u.ReceiveMessage(msgRaw)
	}

	wg.Wait()
}

func (u User) RemoveUser() {
	u.messageBroker.RemoveUser(u.user.ID)
}

// @todo consider just writing the error in write channel
func (u User) CreateChatroom(ctx context.Context, msg models.WSChatroomCreateMessage) {
	slog.Info(fmt.Sprintf("creating chatroom: %s", msg.ChatroomName))
	chatroomID, err := u.chatroomNoSQLRepoer.CreateChatroom(
		ctx,
		models.CreateChatroomParams{
			ChatroomName: msg.ChatroomName,
			AddUsers:     msg.InviteUsers,
		},
	)
	if err != nil {
		slog.Error("failed to create chatroom ", err)
		return
	}

	if err := u.EnterChatroom(ctx, chatroomID, true); err != nil {
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
		return
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

	u.messageBroker.DistributeMessage(ctx, updateMessage)
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

	u.messageBroker.DistributeMessage(ctx, deleteMessage)
}

func (u User) InitialConnect(ctx context.Context) error {
	chatroomIDs, err := u.chatroomNoSQLRepoer.GetUserConnectedChatrooms(ctx, u.user.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to find initial connects for user %s", u.user.ID)
	}

	var addUser bool
	if len(chatroomIDs) == 0 {
		slog.Info("no currently connected connecting to mainChat")
		chatroomIDs = append(chatroomIDs, models.MainChatUUID)
		addUser = true
	}

	for i := range chatroomIDs {
		slog.Info(fmt.Sprintf("adding user to chatroom %s", chatroomIDs[i].String()))
		if err := u.EnterChatroom(ctx, chatroomIDs[i], addUser); err != nil {
			return errors.Wrapf(err, "failed to enter chatroom %s", chatroomIDs[i].String())
		}
	}

	slog.Info("finished initial connect")

	return nil
}
