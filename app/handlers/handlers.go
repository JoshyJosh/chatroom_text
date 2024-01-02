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

	"github.com/gin-gonic/gin"
	kratosClient "github.com/ory/kratos-client-go"
	"golang.org/x/exp/slog"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

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

	hasActiveSession(c, logger)

	if pusher, ok := c.Writer.(http.Pusher); ok {
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

func hasActiveSession(ctx context.Context, logger *slog.Logger) {
	logger.Info("testing 123 testing")
	configuration := kratosClient.NewConfiguration()
	configuration.Servers = []kratosClient.ServerConfiguration{
		{
			URL: "http://ory_proxy:4000/.ory", // Kratos Admin API
		},
	}
	kratosAPI := kratosClient.NewAPIClient(configuration)
	resp, r, err := kratosAPI.FrontendApi.ToSession(context.Background()).Cookie("ory_Kratos_session").Execute()
	if err != nil {
		panic(err)
	}

	logger.Info(fmt.Sprintf("resp: %#v\n r: %#v\n", resp, r))
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

	wsConn, err := websocket.Accept(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("failed to accept websocket: %s", err)
	}
	defer wsConn.Close(websocket.StatusInternalError, "closing connection")

	writeChan := make(chan []byte, 10)
	readChan := make(chan []byte)
	closeChan := make(chan struct{})

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	userService, err := chatroom.GetUserServicer(writeChan)
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

	// @todo error handling and graceful exit

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
