package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"chatroom_text/internal/handlers"
	"chatroom_text/internal/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

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

	srv := &http.Server{
		Addr:         httpPort,
		Handler:      router,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}

	log.Fatal(srv.ListenAndServe())
}
