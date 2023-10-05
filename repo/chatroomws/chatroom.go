package chatroomws

import (
	"encoding/json"
	"fmt"
	"sync"

	"chatroom_text/models"
	"chatroom_text/services"

	"github.com/google/uuid"
	"golang.org/x/exp/slog"
)

type chatroomRoster struct {
	clientMap *sync.Map
}

var chatroom chatroomRoster = chatroomRoster{
	clientMap: &sync.Map{},
}

func GetChatroom() *chatroomRoster {
	return &chatroom
}

func (c chatroomRoster) AddNewUser(user services.UserServicer) string {
	id := uuid.New()

	for {
		if _, exists := c.clientMap.LoadOrStore(id, user); exists {
			id = uuid.New()
			continue
		}

		break
	}

	return id.String()
}

func (c chatroomRoster) AddUser(id string, user services.UserServicer) error {
	// Account for possible ID duplicates.
	if _, exists := c.clientMap.LoadOrStore(id, user); !exists {
		return fmt.Errorf("clientMap does not have reserved id %s", id)
	}

	return nil
}

func (c *chatroomRoster) ReceiveMessage(msg models.WSMessage) {
	// @todo add message to chatroom message history
	slog.Info("received message")
	msgRaw, err := json.Marshal(msg)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to unmarshal received message: %v", msg))
		return
	}
	c.DistributeMessage(msgRaw)
}

func (c *chatroomRoster) DistributeMessage(message []byte) {
	slog.Info("distributing message")
	c.clientMap.Range(func(key, user any) bool {
		u, ok := user.(services.UserServicer)
		if !ok {
			slog.Error(fmt.Sprintf("failed to convert client %s in map range, client type %T", key, user))
		}

		u.GetWriteChan() <- message
		return true
	})
}
