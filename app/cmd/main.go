package main

import (
	"flag"
	"fmt"
	"os"

	"chatroom_text/handlers"
	"chatroom_text/middleware"

	"chatroom_text/repo/db"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

func init() {
	if err := db.InitDB(); err != nil {
		panic(err)
	}
}

func main() {
	portFlag := flag.Int("port", 8080, "Listen address")
	flag.Parse()

	httpPort := fmt.Sprintf(":%d", *portFlag)

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

	router := gin.Default()
	authMiddleware := middleware.GetAuthClient(logger)

	router.Use(authMiddleware.SessionMiddleware())

	userHandler := handlers.GetUserHandle()
	router.Static("/static", "./static")
	router.GET("/", userHandler.EnterChat)
	router.GET("/websocket/", userHandler.ConnectWebSocket)

	if err := router.RunTLS(httpPort, "cert.pem", "key.pem"); err != nil {
		slog.Error(fmt.Sprintf("%v", err))
	}
}
