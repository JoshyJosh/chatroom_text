package chatroom

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	chatroomws "chatroom_text/repo/chatroomws"
	services "chatroom_text/services"
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/exp/slog"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type User struct {
	ID           string
	Conn         *websocket.Conn
	WriteChan    chan []byte
	chatroomRepo repo.ChatroomRepoer
}

func GetUserServicer(conn *websocket.Conn) services.UserServicer {
	user := User{
		Conn:         conn,
		WriteChan:    make(chan []byte, 10),
		chatroomRepo: chatroomws.GetChatroom(),
	}

	id := user.chatroomRepo.AddNewUser(user)

	user.ID = id

	return user
}

func (u User) GetWriteChan() chan []byte {
	return u.WriteChan
}

func (u User) ReadLoop(ctx context.Context) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var wsm models.WSMessage
	for {
		if err := wsjson.Read(ctx, u.Conn, &wsm); err != nil {
			logger.Error("Failed to read json: %v", err)
			break
		}

		logger.Info(fmt.Sprintf("received: %s", wsm.Text))

		wsm.Timestamp = time.Now()
		wsm.ClientID = u.ID

		u.chatroomRepo.ReceiveMessage(wsm)
	}
}

func (u User) WriteLoop(ctx context.Context) {
	for msg := range u.WriteChan {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		u.Conn.Write(ctx, websocket.MessageText, msg)
	}
}
