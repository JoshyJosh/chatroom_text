package chatroomws

import (
	"encoding/json"
	"fmt"
	"sync"

	"chatroom_text/models"
	"chatroom_text/repo"

	"golang.org/x/exp/slog"
)

type chatroomRoster struct {
	clientMap *sync.Map
}

var chatroom chatroomRoster = chatroomRoster{
	clientMap: &sync.Map{},
}

func GetChatroomRepoer() repo.ChatroomRepoer {
	return &chatroom
}

func (c chatroomRoster) AddUser(user models.UserWS) error {
	if _, exists := c.clientMap.LoadOrStore(user.ID, user); exists {
		return fmt.Errorf("clientMap already has user with ID %s", user.ID)
	}

	return nil
}

func (c chatroomRoster) RemoveUser(id string) {
	c.clientMap.Delete(id)
}

func (c *chatroomRoster) ReceiveMessage(msg models.WSMessage) {
	// @todo add message to chatroom message history
	slog.Info(fmt.Sprintf("received message: %s", msg.Text))
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
		u, ok := user.(models.UserWS)
		if !ok {
			slog.Error(fmt.Sprintf("failed to convert client %s in map range, client type %T", key, user))
		}

		u.WriteChan <- message
		return true
	})
}
