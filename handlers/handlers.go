package handlers

import (
	"chatroom_text/services/chatroom"
	"net/http"
	"os"
	"sync"

	"golang.org/x/exp/slog"
	"nhooyr.io/websocket"
)

type userHandle struct{}

func GetUserHandle() userHandle {
	return userHandle{}
}

// EnterChat adds users to chatroom.
func (userHandle) EnterChat(w http.ResponseWriter, r *http.Request) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil).WithAttrs(
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

func (userHandle) ConnectWebSocket(w http.ResponseWriter, r *http.Request) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil).WithAttrs(
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
		logger.Error("Failed to accept websocket: %v", err)
	}
	defer c.Close(websocket.StatusInternalError, "closing connection")

	user := chatroom.GetUserServicer(c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		user.WriteLoop(r.Context())
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		user.ReadLoop(r.Context())
	}()

	wg.Wait()
}
