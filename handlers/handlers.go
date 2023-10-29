package handlers

import (
	"chatroom_text/models"
	"chatroom_text/services"
	"chatroom_text/services/chatroom"
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/exp/slog"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type userHandle struct{}

func GetUserHandle() userHandle {
	return userHandle{}
}

// EnterChat adds users to chatroom.
func (userHandle) EnterChat(w http.ResponseWriter, r *http.Request) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil).WithAttrs(
		[]slog.Attr{
			{
				Key:   "stage",
				Value: slog.StringValue("enterChat"),
			},
		},
	))

	if pusher, ok := w.(http.Pusher); ok {
		logger.Info("pushed http2")

		options := &http.PushOptions{
			Header: http.Header{
				"Accept-Encoding": r.Header["Accept-Encoding"],
			},
		}

		if err := pusher.Push("/static/index.js", options); err != nil {
			logger.Error("Failed to push: %w", err)
		}
	}

	http.ServeFile(w, r, "static/index.html")
}

type userWebsocketHandle struct {
	conn        *websocket.Conn
	readChan    chan []byte
	writeChan   chan []byte
	closeChan   chan struct{}
	userService services.UserServicer
}

func (userHandle) ConnectWebSocket(w http.ResponseWriter, r *http.Request) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil).WithAttrs(
		[]slog.Attr{
			{
				Key:   "websocket",
				Value: slog.StringValue("enterChat"),
			},
		},
	))
	logger.Info("connecting websocket")
	defer logger.Info("exiting websocket")

	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		logger.Error("failed to accept websocket: %v", err)
	}
	defer c.Close(websocket.StatusInternalError, "closing connection")

	writeChan := make(chan []byte, 10)
	readChan := make(chan []byte)
	closeChan := make(chan struct{})

	ctx, cancel := context.WithCancel(r.Context())

	userService, err := chatroom.GetUserServicer(writeChan)
	if err != nil {
		logger.Error(err.Error())

		if err := c.Write(ctx, websocket.MessageText, []byte(`{"err":"failed to add user to chatroom"}`)); err != nil {
			logger.Error(err.Error())
		}
		return
	}
	defer userService.RemoveUser()

	user := userWebsocketHandle{
		conn:        c,
		readChan:    readChan,
		writeChan:   writeChan,
		closeChan:   closeChan,
		userService: userService,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		user.WriteLoop(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := user.ReadLoop(ctx); err != nil {
			cancel()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := userService.EnterChatroom()
		if err != nil {
			cancel()
		}
	}()

	// @todo error handling adn graceful exit

	wg.Wait()
}

func (u userWebsocketHandle) WriteLoop(ctx context.Context) {
	defer slog.Info("exiting writeloop")
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-u.writeChan:
			u.writeMsg(ctx, msg)
		}
	}
}

func (u userWebsocketHandle) writeMsg(ctx context.Context, msg []byte) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := u.conn.Write(ctx, websocket.MessageText, msg); err != nil {
		slog.Error(err.Error())
	}
}

func (u userWebsocketHandle) ReadLoop(ctx context.Context) error {
	var msg models.WSMessage
	for {
		if err := wsjson.Read(ctx, u.conn, &msg); err != nil {
			slog.Info(fmt.Sprintf("got error status: %s", websocket.CloseStatus(err).String()))
			if websocket.CloseStatus(err) == websocket.StatusAbnormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return err
			}

			slog.Error("Failed to read json: %v", err)
			// @todo consider retry
			continue
		}

		slog.Info(fmt.Sprintf("received message: %s", msg.Text))

		u.userService.ReadMessage(msg)
	}
}
