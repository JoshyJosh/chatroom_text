package handlers

import (
	"chatroom_text/internal/middleware"
	"chatroom_text/internal/models"
	"chatroom_text/internal/services"
	"chatroom_text/internal/services/chatroom"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Connmap is used to limit one connection per user.
var wsConnMap sync.Map

func init() {
	wsConnMap = sync.Map{}
}

type userHandle struct{}

func GetUserHandle() userHandle {
	return userHandle{}
}

// EnterChat adds users to chatroom.
func (userHandle) EnterChat(c *gin.Context) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil).WithAttrs(
		[]slog.Attr{
			{
				Key:   "stage",
				Value: slog.StringValue("enterChat"),
			},
		},
	))

	if pusher := c.Writer.Pusher(); pusher != nil {
		logger.Info("pushed http2")

		options := &http.PushOptions{
			Header: http.Header{
				"Accept-Encoding": []string{c.GetHeader("Accept-Encoding")},
			},
		}

		if err := pusher.Push("/static/index.js", options); err != nil {
			logger.Error("Failed to push: %w", err)
		}
	}

	http.ServeFile(c.Writer, c.Request, "static/index.html")
}

type userWebsocketHandle struct {
	conn        *websocket.Conn
	readChan    chan []byte
	writeChan   chan []byte
	closeChan   chan struct{}
	userService services.UserServicer
}

func (userHandle) ConnectWebSocket(c *gin.Context) {
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

	userData, err := middleware.GetAuthClient(logger).GetUserData(c.Request.Context(), c.Request)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to get user data: %s", err))
		return
	}

	// @todo uncomment after gracefull disconnect is fixed.
	// if _, ok := wsConnMap.Load(userData.ID); ok {
	// 	slog.Error("duplicate ws connection  in wsConnMap")
	// 	return
	// }

	wsConn, err := websocket.Accept(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("failed to accept websocket: %s", err)
	}
	defer wsConn.Close(websocket.StatusInternalError, "closing connection")

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	wsConnMap.Store(userData.ID, struct{}{})
	defer func() {
		// @todo does not delete every time, need graceful reconnect
		slog.Info("deleting wsConnMap entry")
		wsConnMap.Delete(userData.ID)
		slog.Info("deleted wsConnMap entry")
	}()

	writeChan := make(chan []byte, 10)
	readChan := make(chan []byte)
	closeChan := make(chan struct{})

	userService, err := chatroom.GetUserServicer(ctx, writeChan, userData)
	if err != nil {
		logger.Error(err.Error())

		if err := wsConn.Write(ctx, websocket.MessageText, []byte(`{"err":"failed to add user to chatroom"}`)); err != nil {
			logger.Error(err.Error())
		}
		return
	}
	defer userService.RemoveUser()

	user := userWebsocketHandle{
		conn:        wsConn,
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
			logger.Error("failed to read from read loop", err)
			cancel()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := user.Healthcheck(ctx); err != nil {
			logger.Error("failed to get pong from healthcheck", err)
			cancel()
		}
	}()

	// Start TextMessageListener
	wg.Add(1)
	go func() {
		defer wg.Done()
		userService.ListenForMessages(ctx)
	}()

	// Entering main chat
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := userService.InitialConnect(ctx); err != nil {
			logger.Error("failed to enter chatroom", err)
			cancel()
		}
	}()

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

	slog.Info(fmt.Sprintf("writing msg: %s", msg))

	if err := u.conn.Write(ctx, websocket.MessageText, msg); err != nil {
		slog.Error(err.Error())
	}
}

func (u userWebsocketHandle) ReadLoop(ctx context.Context) error {
	for {
		var msg models.WSMessage
		if err := wsjson.Read(ctx, u.conn, &msg); err != nil {
			if websocket.CloseStatus(err) == websocket.StatusAbnormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				slog.Info(fmt.Sprintf("got error status: %s", websocket.CloseStatus(err).String()))
				return err
			}

			// Error code -1 is a non-close status.
			if websocket.CloseStatus(err) == -1 {
				slog.Debug("unknown error: ", err.Error())
				continue
			}

			if strings.Contains(err.Error(), "WebSocket closed") {
				slog.Error(fmt.Sprintf("Got generic close status for websocket closed, error: %s", err))
				return err
			}

			slog.Error("Failed to read json: %v", err)

			continue
		}

		switch {
		case msg.TextMessage != nil:
			u.userService.SendMessage(ctx, *msg.TextMessage)
		case msg.ChatroomMessage != nil:
			switch {
			case msg.ChatroomMessage.Create != nil:
				// @todo reconsider having return values since it can be propagted via write channel
				u.userService.CreateChatroom(ctx, *msg.ChatroomMessage.Create)
			case msg.ChatroomMessage.Update != nil:
				u.userService.UpdateChatroom(ctx, *msg.ChatroomMessage.Update)
			case msg.ChatroomMessage.Delete != nil:
				u.userService.DeleteChatroom(ctx, *msg.ChatroomMessage.Delete)
			}
		default:
			slog.Error("Received unknown message")
		}
	}
}

// Healthcheck is used to still keep an idle websocket connnection live.
func (u userWebsocketHandle) Healthcheck(ctx context.Context) error {
	// Websocket timeout is usually 1 minute, however this should leave a 5 second tolerance.
	ticker := time.NewTicker(time.Minute - 5*time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := u.conn.Ping(ctx); err != nil {
				return errors.Wrap(err, "failed to ping client")
			}
		case <-ctx.Done():
			return nil
		}
	}
}
