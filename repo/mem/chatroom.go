package mem

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"chatroom_text/models"
	"chatroom_text/repo"

	"golang.org/x/exp/slog"
)

type chatroomRoster struct {
	userMap *sync.Map
	logs    []models.ChatroomLog
}

var chatroom chatroomRoster = chatroomRoster{
	userMap: &sync.Map{},
	logs:    []models.ChatroomLog{},
}

func GetChatroomRepoer() repo.ChatroomRepoer {
	return &chatroom
}

func (c chatroomRoster) AddUser(user models.User) error {
	if user.ID == "" {
		return errors.New("tried to add user with empty ID")
	}

	if _, exists := c.userMap.LoadOrStore(user.ID, user); exists {
		return fmt.Errorf("clientMap already has user with ID %s", user.ID)
	}

	return nil
}

func (c chatroomRoster) RemoveUser(id string) {
	c.userMap.Delete(id)
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

func (c *chatroomRoster) DistributeMessage(msgRaw []byte) {
	slog.Info("distributing message")

	c.userMap.Range(func(key, user any) bool {
		u, ok := user.(models.User)
		if !ok {
			slog.Error(fmt.Sprintf("failed to convert client %s in map range, client type %T", key, user))
		}

		u.WriteChan <- msgRaw
		return true
	})
}
