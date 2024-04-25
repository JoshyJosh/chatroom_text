package mem

import (
	"chatroom_text/internal/models"
	"chatroom_text/internal/repo"
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

// @todo consider if it is needed to have a roster of all connected users on a user basis
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
