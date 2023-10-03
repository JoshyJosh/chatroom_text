package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/slog"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type WSMessage struct {
	Text      string    `json:"msg"`
	Timestamp time.Time `json:"timestamp"`
	ClientID  string    `json:"clientID"` // should corelate with WSClient ID
}

type WSClient struct {
	ID        string
	Conn      *websocket.Conn
	WriteChan chan []byte
}

type Chatroom struct {
	clientMap sync.Map
}

var chatroom Chatroom = Chatroom{
	clientMap: sync.Map{},
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
	defer c.Close(websocket.StatusInternalError, "closing connection")

	wsClient := WSClient{
		Conn:      c,
		WriteChan: make(chan []byte, 10),
	}

	defer close(wsClient.WriteChan)

	chatroom.addClient(&wsClient)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var wsm WSMessage
		for {
			if err := wsjson.Read(r.Context(), c, &wsm); err != nil {
				logger.Error("Failed to read json: %v", err)
				break
			}

			logger.Info(fmt.Sprintf("received: %s", wsm.Text))

			wsm.Timestamp = time.Now()
			fmt.Println(wsClient.ID)
			wsm.ClientID = wsClient.ID

			chatroom.receiveMessage(wsm)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for msg := range wsClient.WriteChan {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			wsClient.Conn.Write(ctx, websocket.MessageText, msg)
		}
	}()

	wg.Wait()
}

func (c *Chatroom) addClient(client *WSClient) {
	id := uuid.New()
	defer func() {
		client.ID = id.String()
	}()

	// Account for possible ID duplicates.
	for {
		if _, exists := c.clientMap.LoadOrStore(id, *client); exists {
			id = uuid.New()
			continue
		}

		return
	}
}

func (c *Chatroom) distributeMessage(message []byte) {
	slog.Info("distributing message")
	c.clientMap.Range(func(key, client any) bool {
		c, ok := client.(WSClient)
		if !ok {
			slog.Error(fmt.Sprintf("failed to convert client %s in map range, client type %T", key, client))
		}

		c.WriteChan <- message
		return true
	})
}

func (c *Chatroom) receiveMessage(msg WSMessage) {
	// @todo add message to chatroom message history
	slog.Info("received message")
	msgRaw, err := json.Marshal(msg)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to unmarshal received message: %v", msg))
		return
	}
	c.distributeMessage(msgRaw)
}
