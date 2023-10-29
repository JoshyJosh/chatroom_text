package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"chatroom_text/handlers"

	"chatroom_text/repo/db"

	"golang.org/x/exp/slog"
)

func init() {
	if err := db.InitDB(); err != nil {
		panic(err)
	}
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

	userHandler := handlers.GetUserHandle()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", userHandler.EnterChat)
	http.HandleFunc("/websocket/", userHandler.ConnectWebSocket)

	if err := http.ListenAndServeTLS(httpPort, "cert.pem", "key.pem", nil); err != nil {
		slog.Error(fmt.Sprintf("%v", err))
	}
}
