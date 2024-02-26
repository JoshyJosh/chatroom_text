package middleware

import (
	"chatroom_text/models"
	"context"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/exp/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ory "github.com/ory/client-go"
	"github.com/pkg/errors"
)

type kratosMiddleware struct {
	Client *ory.APIClient
	log    *slog.Logger
}

func GetAuthClient(logger *slog.Logger) *kratosMiddleware {
	return newAuthClient(logger)
}

func newAuthClient(logger *slog.Logger) *kratosMiddleware {
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
		session, err := k.getOrySession(c, c.Request)
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

func (k *kratosMiddleware) getOrySession(ctx context.Context, r *http.Request) (*ory.Session, error) {
	cookie, err := r.Cookie("ory_kratos_session")
	if err != nil {
		return nil, err
	}
	if cookie == nil {
		return nil, errors.New("no session found in cookie")
	}

	session, _, err := k.Client.FrontendAPI.ToSession(ctx).Cookie(cookie.String()).Execute()
	if err != nil {
		return nil, err
	}

	return session, err
}

func (k *kratosMiddleware) GetUserData(ctx context.Context, r *http.Request) (models.AuthUserData, error) {
	var userData models.AuthUserData
	session, err := k.getOrySession(ctx, r)
	if err != nil {
		return userData, err
	}

	userData.ID, err = uuid.Parse(session.Identity.Id)
	if err != nil {
		return userData, errors.Wrap(err, "failed to parse auth identity ID")
	}
	traits := session.Identity.Traits.(map[string]interface{})
	nameMapRaw, ok := traits["name"]
	if !ok {
		return userData, errors.New("failed to find name")
	}

	nameMap := nameMapRaw.(map[string]interface{})

	userData.Name, ok = nameMap["username"].(string)
	if !ok {
		return userData, errors.New("failed to find username")
	}

	slog.Debug(fmt.Sprintf("Got user data: %#v", userData))

	return userData, nil
}
