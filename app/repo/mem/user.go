package mem

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/exp/slog"
)

type userWSRoster struct {
	userIDMap *sync.Map
}

var userRoster userWSRoster = userWSRoster{
	userIDMap: &sync.Map{},
}

func GetUserRepoer() repo.UserRepoer {
	return userRoster
}

func (u userWSRoster) AddUser(user models.User) error {
	slog.Info("adding user to roster")

	if _, exists := u.userIDMap.LoadOrStore(user.ID, user); exists {
		return fmt.Errorf("failed to add user with %s to userWSRoster", user.ID.String())
	}

	slog.Info(fmt.Sprintf("added user with ID: %s", user.ID))
	return nil
}

func (u userWSRoster) RemoveID(id uuid.UUID) {
	u.userIDMap.Delete(id)
}
