package chatroom

import (
	"bytes"
	"chatroom_text/models"
	"context"
	"regexp"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type MockNoSQL struct{}

func (MockNoSQL) SelectChatroomLogs(ctx context.Context, params models.SelectDBMessagesParams) ([]models.ChatroomLog, error) {
	return nil, nil
}
func (MockNoSQL) InsertChatroomLogs(ctx context.Context, params models.InsertDBMessagesParams) error {
	return nil
}
func (MockNoSQL) CreateChatroom(ctx context.Context, params models.CreateChatroomParams) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}
func (MockNoSQL) UpdateChatroom(ctx context.Context, chatroomID uuid.UUID, newName string, addUsers []string, removeUsers []string) error {
	return nil
}
func (MockNoSQL) DeleteChatroom(ctx context.Context, chatroomID uuid.UUID) error {
	return nil
}
func (MockNoSQL) GetChatroomEntry(ctx context.Context, chatroomID uuid.UUID) (models.ChatroomEntry, error) {
	return models.ChatroomEntry{}, nil
}
func (MockNoSQL) AddUserToChatroom(ctx context.Context, chatroomID uuid.UUID, userID uuid.UUID) error {
	return nil
}
func (MockNoSQL) GetUserConnectedChatrooms(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

type MockUserRepo struct{}

func (MockUserRepo) AddUser(user models.User) error {
	return nil
}

// ReceiveMessage receives message from user to chatroom.
func (MockUserRepo) RemoveID(id uuid.UUID) {}

func newUsersService() User {
	return User{
		user: models.User{
			ID:        uuid.MustParse("6c665468-6fc4-487e-9a5a-1a4c271ec698"),
			Name:      "testuser",
			WriteChan: make(chan []byte),
		},
		userRepo:            MockUserRepo{},
		chatroomRepos:       &sync.Map{}, // map which key is the chat uuid and value is repo.ChatroomRepoer
		chatroomNoSQLRepoer: MockNoSQL{},
	}
}

func TestSuccessfullyEnterMainChatroom(t *testing.T) {
	a := assert.New(t)
	uService := newUsersService()
	receivedMessages := [][]byte{}

	var wg sync.WaitGroup
	methodDone := make(chan struct{}, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case msg := <-uService.user.WriteChan:
				receivedMessages = append(receivedMessages, msg)
			case <-methodDone:
				return
			}
		}
	}()

	err := uService.EnterChatroom(context.Background(), models.MainChatUUID, false)
	a.NoError(err)

	methodDone <- struct{}{}
	wg.Wait()

	expectedMessages := [][]byte{
		[]byte(`{"chatroom":{"enter":{"chatroomName":"","chatroomID":"00000000-0000-0000-0000-000000000000"}}}`),
		[]byte(`{"text":{"msg":"entered chat","timestamp":"","userID":"6c665468-6fc4-487e-9a5a-1a4c271ec698","userName":"testuser","chatroomID":"00000000-0000-0000-0000-000000000001"}}`),
	}

	re := regexp.MustCompile(`("timestamp":")([\d-T:.+]+)`)
	for i := range receivedMessages {
		var receivedMessage []byte
		if i == 1 {
			timestamp := re.FindSubmatch(receivedMessages[i])
			receivedMessage = bytes.ReplaceAll(receivedMessages[i], timestamp[2], []byte(""))
		} else {
			receivedMessage = receivedMessages[i]
		}
		a.Equal(expectedMessages[i], receivedMessage)
	}
}
