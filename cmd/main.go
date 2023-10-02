package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/exp/slog"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type WSMessage struct {
	Text string `json:"msg"`
}

func main() {
	httpPort := fmt.Sprintf(":%s", *(flag.String("port", "8080", "Listen address")))

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil).WithAttrs(
		[]slog.Attr{
			{
				Key:   "stage",
				Value: slog.StringValue("main"),
			},
		},
	))
	logger.Info("starting")
	defer logger.Info("stopping")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", enterChat)
	http.HandleFunc("/websocket/", connectWebSocket)

	if err := http.ListenAndServeTLS(httpPort, "cert.pem", "key.pem", nil); err != nil {
		slog.Error(fmt.Sprintf("%v", err))
	}
}

func enterChat(w http.ResponseWriter, r *http.Request) {
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

func connectWebSocket(w http.ResponseWriter, r *http.Request) {
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
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	var wsm WSMessage
	err = wsjson.Read(ctx, c, &wsm)
	if err != nil {
		logger.Error("Failed to read json: %v", err)
	}

	logger.Info(fmt.Sprintf("received: %s", wsm.Text))

	if err := c.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf("received text: %s", wsm.Text))); err != nil {
		logger.Error("Failed to write message: %v", err)
	}

	c.Close(websocket.StatusNormalClosure, "")
}

func sendMessagefunc(w http.ResponseWriter, r *http.Request) {
	// @todo make send message func
}
