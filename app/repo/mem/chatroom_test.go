package mem

import (
	"bytes"
	"chatroom_text/models"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChatroomRosterAddUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	testChatroomRoster := chatroomRoster{
		userMap: &sync.Map{},
	}

	user1 := models.User{}

	err := testChatroomRoster.AddUser(user1)
	assert.NotNil(err, "expected error for missing ID")
	expectedErr := errors.New("tried to add user with empty ID")
	assert.Equal(expectedErr, err, "unexpected error for missing ID")

	user1.ID = uuid.New()
	err = testChatroomRoster.AddUser(user1)
	assert.Nil(err, "unexpected error when adding user")

	mapLen := 0
	testChatroomRoster.userMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(1, mapLen, "expected one entry in userMap")

	user2 := models.User{
		ID: user1.ID,
	}

	err = testChatroomRoster.AddUser(user2)
	assert.NotNil(err, "expected error for missing ID")
	expectedErr = fmt.Errorf("clientMap already has user with ID %s", user1.ID)
	assert.Equal(expectedErr, err, "unexpected error for duplicate user iD")
}

func TestChatroomRosterUserDistribution(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	testChatroomRoster := chatroomRoster{
		userMap: &sync.Map{},
	}

	user1 := models.User{
		ID:        uuid.New(),
		WriteChan: make(chan []byte, 10),
	}

	// make sure user1 ID is not same as user2
	user2ID := uuid.New()
	for user2ID == user1.ID {
		user2ID = uuid.New()
	}

	user2 := models.User{
		ID:        user2ID,
		WriteChan: make(chan []byte, 10),
	}

	err := testChatroomRoster.AddUser(user1)
	assert.Nil(err, "expected no error for adding user1")
	err = testChatroomRoster.AddUser(user2)
	assert.Nil(err, "expected no error for adding user2")

	mapLen := 0
	testChatroomRoster.userMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(2, mapLen, "expected one entry in userMap")

	curTime := time.Now()
	msg := models.WSTextMessage{
		Text:      "test message",
		Timestamp: curTime,
		UserID:    user1.ID.String(),
	}

	msgRaw, err := json.Marshal(msg)
	assert.Nil(err, "expected no error in WSMessage marshalling")

	var wg sync.WaitGroup

	user1Counter := 0
	user1Msg := bytes.Buffer{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for m := range user1.WriteChan {
			user1Counter++
			user1Msg.Write(m)
		}
	}()

	user2Counter := 0
	user2Msg := bytes.Buffer{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for m := range user2.WriteChan {
			user2Counter++
			user2Msg.Write(m)
		}
	}()
	testChatroomRoster.ReceiveMessage(msg)

	close(user1.WriteChan)
	close(user2.WriteChan)

	wg.Wait()

	assert.Equal(2, user1Counter+user2Counter, "expected message to be distributed to 2 users")
	assert.Equal(msgRaw, user1Msg.Bytes(), "expected received message to be equal")
	assert.Equal(msgRaw, user2Msg.Bytes(), "expected received message to be equal")
}
