package handlers

import (
	"chatroom_text/internal/middleware"
	"chatroom_text/internal/models"
	"chatroom_text/internal/services"
	"chatroom_text/internal/services/chatroom"
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	logger := log.WithField(
		"stage", "EnterChat",
	)

	if pusher := c.Writer.Pusher(); pusher != nil {
		logger.Debug("pushed http2")

		options := &http.PushOptions{
			Header: http.Header{
				"Accept-Encoding": []string{c.GetHeader("Accept-Encoding")},
			},
		}

		if err := pusher.Push("/static/index.js", options); err != nil {
			logger.Error("Failed to push http2: ", err)
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
	logger      *log.Entry
}

func (userHandle) ConnectWebSocket(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"stage": "ConnectWebsocket",
	})
	logger.Info("connecting websocket")
	defer logger.Info("exiting websocket")

	userData, err := middleware.GetAuthClient(logger).GetUserData(c.Request.Context(), c.Request)
	if err != nil {
		logger.Error("failed to get user data: ", err)
		return
	}

	logger = logger.WithField("user_id", userData.ID)

	// @todo uncomment after gracefull disconnect is fixed.
	// if _, ok := wsConnMap.Load(userData.ID); ok {
	// 	logger.Error("duplicate ws connection  in wsConnMap")
	// 	return
	// }

	wsConn, err := websocket.Accept(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("failed to accept websocket: ", err)
	}
	defer wsConn.Close(websocket.StatusInternalError, "closing connection")

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	wsConnMap.Store(userData.ID, struct{}{})
	defer func() {
		// @todo does not delete every time, need graceful reconnect
		logger.Debug("deleting wsConnMap entry")
		wsConnMap.Delete(userData.ID)
		logger.Debug("deleted wsConnMap entry")
	}()

	writeChan := make(chan []byte, 10)
	readChan := make(chan []byte)
	closeChan := make(chan struct{})

	userService, err := chatroom.GetUserServicer(ctx, writeChan, userData)
	if err != nil {
		logger.Error("failed to get user service: ", err)

		if err := wsConn.Write(ctx, websocket.MessageText, []byte(`{"err":"failed to add user to chatroom"}`)); err != nil {
			logger.Error("failed to send user error message", err)
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
		logger:      logger,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		user.writeLoop(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := user.readLoop(ctx); err != nil {
			logger.Error("failed to read from read loop", err)
			cancel()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := user.healthcheck(ctx); err != nil {
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

func (u userWebsocketHandle) writeLoop(ctx context.Context) {
	defer u.logger.Debug("exiting writeloop")
	u.logger.Debug("entering writeloop")

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

	u.logger.Debugf("writing msg: %s", msg)

	if err := u.conn.Write(ctx, websocket.MessageText, msg); err != nil {
		u.logger.Error(err)
	}
}

func (u userWebsocketHandle) readLoop(ctx context.Context) error {
	for {
		var msg models.WSMessage
		if err := wsjson.Read(ctx, u.conn, &msg); err != nil {
			if websocket.CloseStatus(err) == websocket.StatusAbnormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				u.logger.Infof("got error status: %s", websocket.CloseStatus(err).String())
				return err
			}

			// Error code -1 is a non-close status.
			if websocket.CloseStatus(err) == -1 {
				u.logger.Debug("unknown error: ", err.Error())
				continue
			}

			if strings.Contains(err.Error(), "WebSocket closed") {
				u.logger.Error("Got generic close status for websocket closed, error: ", err)
				return err
			}

			u.logger.Error("Failed to read json: ", err)

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
			u.logger.Error("Received unknown message")
		}
	}
}

// Healthcheck is used to still keep an idle websocket connnection live.
func (u userWebsocketHandle) healthcheck(ctx context.Context) error {
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
