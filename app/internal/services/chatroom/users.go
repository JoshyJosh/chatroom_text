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
	log "github.com/sirupsen/logrus"
)

type User struct {
	user                models.User
	messageBroker       repo.ChatroomMessageBroker
	chatroomNoSQLRepoer repo.ChatroomLogger
	logger              *log.Entry
}

func GetUserServicer(ctx context.Context, writeChan chan []byte, userData models.AuthUserData) (services.UserServicer, error) {
	chatroomNoSQLRepoer, err := chatroomNoSQL.GetChatroomNoSQLRepoer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logger for user servicer")
	}

	logger := log.WithFields(log.Fields{
		"userID": userData.ID,
		"stage":  "userServicer",
	})

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
		logger:              logger,
	}

	return userService, nil
}

func (u User) EnterChatroom(ctx context.Context, chatroomID uuid.UUID) error {
	logger := u.logger.WithField("chatroomID", chatroomID.String())
	logger.Info("entering chatroom")

	if err := u.messageBroker.BindToMessageQueue(chatroomID); err != nil {
		return err
	}

	chatroomEntry, err := u.chatroomNoSQLRepoer.SelectChatroomEntry(ctx, chatroomID)
	if err != nil {
		return errors.Wrap(err, "failed to get chatroom entry")
	}

	// @todo determine if users have actually been added to nosql entry
	// If user connects the first time mainChat adds the user if it is not listed
	u.chatroomNoSQLRepoer.AddUserToChatroom(ctx, chatroomID, u.user.ID)

	users, err := u.chatroomNoSQLRepoer.SelectChatroomUsers(ctx, chatroomID)
	if err != nil {
		return errors.Wrap(err, "failed to select chatroom users")
	}

	userList := []models.WSUserEntry{}
	for _, user := range users {
		userList = append(userList, models.WSUserEntry{
			ID:   user.ID.String(),
			Name: user.Name,
		})
	}

	msgRaw, err := json.Marshal(models.WSMessage{
		ChatroomMessage: &models.ChatroomMessage{
			Enter: &models.WSChatroomEnterMessage{
				ChatroomName: chatroomEntry.Name,
				ChatroomID:   chatroomEntry.ChatroomID.String(),
				UserList:     userList, // @todo stop debugging
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal chatroom entry message")
	}

	u.ReceiveMessage(msgRaw)

	// Get current text history from chatroom.
	logger.Debug("getting logs")
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

	logger.Debug("sending user entry name")
	// @todo distribute user entered in chatroom
	err = u.messageBroker.DistributeUserEntryMessage(
		ctx,
		chatroomID,
		models.ChatroomMessage{
			AddUser: &models.WSChatroomUserEntry{
				ChatroomID: chatroomID.String(),
				User: models.WSUserEntry{
					ID:   u.user.ID.String(),
					Name: u.user.Name,
				},
			},
		},
	)
	if err != nil {
		logger.Error("failed to send user entry: ", err)
		return err
	}

	err = u.messageBroker.BindToUsersQueue(chatroomID)
	if err != nil {
		logger.Error("failed to bind user to roster exchange: ", err)
		return err
	}

	logger.Debug("sending enter message")
	// Send user entry text message to all people in chatroom.
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
	logger := u.logger.WithField("chatroomID", msg.ChatroomID)
	msg.Timestamp = models.StandardizeTime(time.Now())
	msg.UserID = u.user.ID.String()
	msg.UserName = u.user.Name

	chatroomID, err := uuid.Parse(msg.ChatroomID)
	if err != nil {
		logger.Error("failed to parse chatroomID: ", err)
		return
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
		logger.Error("failed to insert chatroom log: ", err)
		return
	}

	msgBytes, err := json.Marshal(models.WSMessage{
		TextMessage: &msg,
	})
	if err != nil {
		// @todo propagate error message to websocket
		logger.Error("failed to marshal chatroom log: ", err)
		return
	}

	u.messageBroker.DistributeMessage(ctx, chatroomID, msgBytes)
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
		u.messageBroker.Listen(ctx, msgBytesChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msgRaw := <-msgBytesChan:
				u.logger.Debug(fmt.Sprintf("sending to receive message: %s", msgRaw))
				u.ReceiveMessage(msgRaw)
			}
		}
	}()

	wg.Wait()
}

func (u User) RemoveUser() {
	u.messageBroker.RemoveUser(u.user.ID)
}

// @todo consider just writing the error in write channel
func (u User) CreateChatroom(ctx context.Context, msg models.WSChatroomCreateMessage) {
	u.logger.Info("creating chatroom: ", msg.ChatroomName)
	chatroomID, err := u.chatroomNoSQLRepoer.CreateChatroom(
		ctx,
		models.CreateChatroomParams{
			ChatroomName: msg.ChatroomName,
			AddUsers:     msg.InviteUsers,
		},
	)

	logger := u.logger.WithField("chatroomID", chatroomID.String())
	if err != nil {
		logger.Error("failed to create chatroom ", err)
		return
	}

	if err := u.EnterChatroom(ctx, chatroomID); err != nil {
		logger.Error("failed to enter chatroom ", err)
		return
	}
}

// @todo implement adding and removing users.
func (u User) UpdateChatroom(ctx context.Context, msg models.WSChatroomUpdateMessage) {
	logger := u.logger.WithField("chatroomID", msg.ChatroomID)
	chatroomID, err := uuid.Parse(msg.ChatroomID)
	if err != nil {
		logger.Error("failed to parse uuid: ", err)
		return
	}

	if models.MainChatUUID == chatroomID {
		logger.Error("cannot update main chat")
		return
	}

	if msg.NewChatroomName == "" {
		logger.Error("cannot set empty name for chatroom")
		return
	}

	if err := u.chatroomNoSQLRepoer.UpdateChatroom(ctx, chatroomID, msg.NewChatroomName, nil, nil); err != nil {
		logger.Error("failed to update chatroom: ", err)
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
		logger.Error("failed to marshal chatroom update message: ", err)
		return
	}

	u.messageBroker.DistributeMessage(ctx, chatroomID, updateMessage)
}

func (u User) DeleteChatroom(ctx context.Context, msg models.WSChatroomDeleteMessage) {
	logger := u.logger.WithField("chatroomID", msg.ChatroomID)
	chatroomID, err := uuid.Parse(msg.ChatroomID)
	if err != nil {
		logger.Error("failed to parse uuid: ", err)
		return
	}

	if models.MainChatUUID == chatroomID {
		logger.Error("cannot delete main chat")
		return
	}

	if err := u.chatroomNoSQLRepoer.DeleteChatroom(ctx, chatroomID); err != nil {
		logger.Error("failed to delete chatroom: ", err)
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
		logger.Error("failed to marshal chatroom delete message: ", err)
		return
	}

	u.messageBroker.DistributeMessage(ctx, chatroomID, deleteMessage)
}

func (u User) InitialConnect(ctx context.Context) error {
	u.logger.Info("starting initial connect")
	defer u.logger.Info("finished initial connect")
	chatroomIDs, err := u.chatroomNoSQLRepoer.SelectUserConnectedChatrooms(ctx, u.user.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to find initial connects for user %s", u.user.ID)
	}

	if err = u.chatroomNoSQLRepoer.StoreUsername(ctx, u.user); err != nil {
		return errors.Wrapf(err, "failed to store username %s", u.user.ID)
	}

	if len(chatroomIDs) == 0 {
		u.logger.Info("no currently connected chatrooms connecting to mainChat")
		chatroomIDs = append(chatroomIDs, models.MainChatUUID)
	}

	for i := range chatroomIDs {
		u.logger.Info(fmt.Sprintf("adding user to chatroom %s", chatroomIDs[i].String()))
		if err := u.EnterChatroom(ctx, chatroomIDs[i]); err != nil {
			return errors.Wrapf(err, "failed to enter chatroom %s", chatroomIDs[i].String())
		}
	}

	return nil
}
