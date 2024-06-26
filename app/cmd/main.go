package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"

	"chatroom_text/internal/handlers"
	"chatroom_text/internal/middleware"
	"chatroom_text/internal/models"
	"chatroom_text/internal/repo/nosql"
	"chatroom_text/internal/repo/rabbitmq"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func main() {
	portFlag := flag.Int("port", 8080, "Listen address")
	flag.Parse()

	prepareJSFiles()

	httpPort := fmt.Sprintf(":%d", *portFlag)

	logger := log.WithFields(log.Fields{
		"stage": "main",
	})

	nosql.InitAddr()

	logger.Info("starting with address: ", *portFlag)
	defer logger.Info("stopping")

	router := gin.Default()
	authMiddleware := middleware.GetAuthClient(logger)

	err := rabbitmq.InitRabbitMQClient()
	if err != nil {
		panic(err)
	}
	defer rabbitmq.CloseRabbitMQClient()

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

func prepareJSFiles() {
	wsHost := os.Getenv("WS_HOST")
	if wsHost == "" {
		panic("missing WS_HOST env variable")
	}
	t, err := template.ParseFiles("./templates/index.js.template")
	if err != nil {
		panic(err)
	}

	data := bytes.NewBuffer([]byte{})
	t.Execute(data, models.JSScriptTemplateData{HostURI: wsHost})

	os.WriteFile("./static/index.js", data.Bytes(), 0644)
}
