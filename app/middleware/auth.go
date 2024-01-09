package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/exp/slog"

	"github.com/gin-gonic/gin"
	ory "github.com/ory/client-go"
)

type kratosMiddleware struct {
	Client *ory.APIClient
	log    *slog.Logger
}

func GetAuthClient(logger *slog.Logger) *kratosMiddleware {
	return newAuthMiddleware(logger)
}

func newAuthMiddleware(logger *slog.Logger) *kratosMiddleware {
	kratosURL := os.Getenv("KRATOS_URL")
	configuration := ory.NewConfiguration()
	configuration.Servers = []ory.ServerConfiguration{
		{
			// URL: "http://kratos:4434", // Kratos Admin API
			URL: kratosURL, // Kratos Admin API
		},
	}
	return &kratosMiddleware{
		Client: ory.NewAPIClient(configuration),
		log:    logger,
	}
}

func (k *kratosMiddleware) SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := k.validateSession(c.Request)
		loginURL := os.Getenv("LOGIN_URL")
		if err != nil {
			fmt.Println("this does not work")
			fmt.Printf("%s\n", err)
			// c.Redirect(http.StatusTemporaryRedirect, "http://127.0.0.1:4455/login")
			c.Redirect(http.StatusTemporaryRedirect, loginURL)
			return
		}

		fmt.Println("got session!!!!")
		if !*session.Active {
			c.Redirect(http.StatusMovedPermanently, "http://127.0.0.1:8080/ping")
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

	resp, _, err := k.Client.FrontendAPI.ToSession(context.Background()).Cookie(cookie.String()).Execute()
	if err != nil {
		return nil, err
	}
	return resp, err
}
