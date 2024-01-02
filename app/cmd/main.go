package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	"chatroom_text/handlers"

	"chatroom_text/repo/db"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"

	ory "github.com/ory/kratos-client-go"
)

func init() {
	if err := db.InitDB(); err != nil {
		panic(err)
	}
}

type kratosMiddleware struct {
	ory *ory.APIClient
}

func newAuthMiddleware() *kratosMiddleware {
	configuration := ory.NewConfiguration()
	configuration.Servers = []ory.ServerConfiguration{
		{
			URL: "http://kratos:4000", // Kratos Admin API
		},
	}
	return &kratosMiddleware{
		ory: ory.NewAPIClient(configuration),
	}
}

func (k *kratosMiddleware) SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := k.validateSession(c.Request)
		if err != nil {
			c.Redirect(http.StatusMovedPermanently, "http://127.0.0.1:4000/.ory/login")
			return
		}
		if !*session.Active {
			c.Redirect(http.StatusMovedPermanently, "https://127.0.0.1:8080")
			return
		}
		c.Next()
	}
}

func (k *kratosMiddleware) validateSession(r *http.Request) (*ory.Session, error) {
	cookie, err := r.Cookie("ory_kratos_session")
	if err != nil {
		return nil, err
	}
	if cookie == nil {
		return nil, errors.New("no session found in cookie")
	}
	resp, _, err := k.ory.FrontendApi.ToSession(context.Background()).Cookie(cookie.String()).Execute()
	if err != nil {
		return nil, err
	}
	return resp, nil
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
	authMiddleware := newAuthMiddleware()

	router.Use(authMiddleware.SessionMiddleware())

	userHandler := handlers.GetUserHandle()
	// router.File("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	router.Static("/static", "./static")
	router.GET("/", userHandler.EnterChat)
	router.GET("/websocket/", userHandler.ConnectWebSocket)

	if err := router.RunTLS(httpPort, "cert.pem", "key.pem"); err != nil {
		slog.Error(fmt.Sprintf("%v", err))
	}
}
