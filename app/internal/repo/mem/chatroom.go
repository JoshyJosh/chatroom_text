package mem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"chatroom_text/internal/models"

	"github.com/google/uuid"
	"golang.org/x/exp/slog"
)

// @todo consider moving to models
type chatroomRoster struct {
	userMap *sync.Map
	logs    []models.ChatroomLog
}

// @todo switch to rabbitMQ in order to have scalability
// map of key uuid and value chatroomRoster pointer
var chatroomMap = &sync.Map{}

var mainchat = chatroomRoster{
	userMap: &sync.Map{},
	logs:    []models.ChatroomLog{},
}

func init() {
	chatroomMap.Store(models.MainChatUUID, &mainchat)
}

func GetChatroomRepoer(chatroomUUID uuid.UUID) *chatroomRoster {
	roster, _ := chatroomMap.LoadOrStore(
		chatroomUUID,
		&chatroomRoster{
			userMap: &sync.Map{},
			logs:    []models.ChatroomLog{},
		},
	)

	return roster.(*chatroomRoster)

}

func (c chatroomRoster) AddUser(user models.User) error {
	if user.ID == [16]byte{} {
		return errors.New("tried to add user with empty ID")
	}

	if _, exists := c.userMap.LoadOrStore(user.ID, user); exists {
		return fmt.Errorf("clientMap already has user with ID %s", user.ID)
	}

	return nil
}

func (c chatroomRoster) RemoveUser(userID uuid.UUID) error {
	c.userMap.Delete(userID)
	return nil
}

func (c *chatroomRoster) ReceiveMessage(msg models.WSTextMessage) error {
	slog.Info(fmt.Sprintf("received message: %s", msg.Text))

	// @todo technically more efficient but make it readable for future implementations
	msgRaw, err := json.Marshal(models.WSMessage{
		TextMessage: &msg,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("failed to unmarshal received message: %v", msg))
		return err
	}

	c.DistributeMessage(context.Background(), msgRaw)

	return nil
}

func (c *chatroomRoster) DistributeMessage(ctx context.Context, msgBytes models.WSTextMessageBytes) error {
	slog.Info("distributing message")

	c.userMap.Range(func(key, user any) bool {
		slog.Info(fmt.Sprintf("Sending to user entry: %v", user))
		u, ok := user.(models.User)
		if !ok {
			slog.Error(fmt.Sprintf("failed to convert client %s in map range, client type %T", key, user))
		}

		u.WriteChan <- msgBytes
		return true
	})

	return nil
}

func (r chatroomRoster) Listen(msgBytesChan chan<- models.WSTextMessageBytes) {
	fmt.Println("blabla")
}
