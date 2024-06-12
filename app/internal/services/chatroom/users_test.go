package chatroom

import (
	"bytes"
	"chatroom_text/internal/models"
	"context"
	"fmt"
	"regexp"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type MockNoSQL struct {
	chatroomMap map[string]models.ChatroomEntry
}

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
func (m MockNoSQL) SelectChatroomEntry(ctx context.Context, chatroomID uuid.UUID) (models.ChatroomEntry, error) {
	chatroomEntry, ok := m.chatroomMap[chatroomID.String()]
	if !ok {
		return models.ChatroomEntry{}, errors.New("failed to find chatroom")
	}
	return chatroomEntry, nil
}
func (MockNoSQL) AddUserToChatroom(ctx context.Context, chatroomID uuid.UUID, userID uuid.UUID) error {
	return nil
}
func (MockNoSQL) SelectChatroomUsers(ctx context.Context, chatroomID uuid.UUID) ([]models.User, error) {
	return nil, nil
}
func (MockNoSQL) SelectUserConnectedChatrooms(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (MockNoSQL) StoreUsername(ctx context.Context, user models.User) error {
	return nil
}

type MockUserRepo struct{}

func (MockUserRepo) AddUser(chatroomID uuid.UUID) error {
	return nil
}

func (MockUserRepo) RemoveUser(chatroomID uuid.UUID) error {
	return nil
}

func (MockUserRepo) DistributeMessage(ctx context.Context, chatroomID uuid.UUID, msgBytes models.WSTextMessageBytes) error {
	return nil
}

func (MockUserRepo) Listen(ctx context.Context, msgBytesChan chan<- models.WSTextMessageBytes) {}

func (MockUserRepo) DistributeUserEntryMessage(ctx context.Context, chatroomID uuid.UUID, msgBytes models.WSUserEntry) error {
	return nil
}

const testChatroomID = "3ee13cbc-e2c3-4975-957f-c40eab28f83d"

func newUsersService() User {
	return User{
		user: models.User{
			ID:        uuid.MustParse("6c665468-6fc4-487e-9a5a-1a4c271ec698"),
			Name:      "testuser",
			WriteChan: make(chan []byte),
		},
		// userRepo:      MockUserRepo{},
		// chatroomRepos: &sync.Map{}, // map which key is the chat uuid and value is repo.ChatroomRepoer
		chatroomNoSQLRepoer: MockNoSQL{
			map[string]models.ChatroomEntry{
				models.MainChatUUID.String(): {
					ChatroomID: models.MainChatUUID,
					Name:       "",
				},
				testChatroomID: {
					ChatroomID: uuid.MustParse(testChatroomID),
					Name:       "testChatroom",
				},
			},
		},
	}
}

func TestSuccessfullyEnterMainChatroom(t *testing.T) {
	a := assert.New(t)
	uService := newUsersService()
	uService.messageBroker = MockUserRepo{}
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

	err := uService.EnterChatroom(context.Background(), models.MainChatUUID)
	a.NoError(err)

	methodDone <- struct{}{}
	wg.Wait()

	expectedMessages := [][]byte{
		[]byte(fmt.Sprintf(`{"chatroom":{"enter":{"chatroomName":"","chatroomID":"%s"}}}`, models.MainChatUUID.String())),
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

func TestSuccessfullyEnterTestChatroom(t *testing.T) {
	a := assert.New(t)
	uService := newUsersService()
	uService.messageBroker = MockUserRepo{}
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

	chatroomUUID := uuid.MustParse(testChatroomID)
	err := uService.EnterChatroom(context.Background(), chatroomUUID)
	a.NoError(err)

	methodDone <- struct{}{}
	wg.Wait()

	expectedMessages := [][]byte{
		[]byte(fmt.Sprintf(`{"chatroom":{"enter":{"chatroomName":"testChatroom","chatroomID":"%s"}}}`, chatroomUUID)),
		[]byte(fmt.Sprintf(`{"text":{"msg":"entered chat","timestamp":"","userID":"6c665468-6fc4-487e-9a5a-1a4c271ec698","userName":"testuser","chatroomID":"%s"}}`, chatroomUUID)),
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

func TestFailToEnterTestChatroom(t *testing.T) {
	a := assert.New(t)
	uService := newUsersService()
	uService.messageBroker = MockUserRepo{}
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

	chatroomUUID := uuid.New()
	err := uService.EnterChatroom(context.Background(), chatroomUUID)
	a.Error(err)

	methodDone <- struct{}{}
	wg.Wait()
}
