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

func (u userWSRoster) AddUser(user models.User) uuid.UUID {
	slog.Info("adding user to roster")
	user.ID = uuid.New()

	for {
		if _, exists := u.userIDMap.LoadOrStore(user.ID, user); exists {
			user.ID = uuid.New()
			continue
		}

		break
	}

	slog.Info(fmt.Sprintf("added user with ID: %s", user.ID))
	return user.ID
}

func (u userWSRoster) RemoveID(id uuid.UUID) {
	u.userIDMap.Delete(id)
}
