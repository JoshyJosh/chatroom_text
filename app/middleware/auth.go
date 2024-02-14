package middleware

import (
	"context"
	"errors"
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
		session, err := k.validateSession(c, c.Request)
		loginURL := os.Getenv("LOGIN_URL")
		if err != nil {
			slog.Info("Redirecting to login")
			c.Redirect(http.StatusTemporaryRedirect, loginURL)
			return
		}

		if !*session.Active {
			slog.Info("inactive session, redirecting back to chatroom")
			c.Redirect(http.StatusMovedPermanently, "https://127.0.0.1/")
			return
		}
		c.Next()
	}
}

func (k *kratosMiddleware) validateSession(ctx context.Context, r *http.Request) (*ory.Session, error) {
	cookie, err := r.Cookie("ory_kratos_session")
	if err != nil {
		return nil, err
	}
	if cookie == nil {
		return nil, errors.New("no session found in cookie")
	}

	resp, _, err := k.Client.FrontendAPI.ToSession(ctx).Cookie(cookie.String()).Execute()
	if err != nil {
		return nil, err
	}
	return resp, err
}
