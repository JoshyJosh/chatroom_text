package mem

import (
	"chatroom_text/models"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserWSRosterAddUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	testUserWSRoster := userWSRoster{
		userIDMap: &sync.Map{},
	}

	user1 := models.User{}
	id1 := testUserWSRoster.AddUser(user1)

	user2 := models.User{}
	id2 := testUserWSRoster.AddUser(user2)

	assert.NotEqual(id1, id2, "id's should be unique")

	mapLen := 0
	testUserWSRoster.userIDMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(2, mapLen, "expected two entries in userIDMap")
}

func TestUserWSRosterRemoveUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	testUserWSRoster := userWSRoster{
		userIDMap: &sync.Map{},
	}

	user := models.User{}
	id := testUserWSRoster.AddUser(user)

	mapLen := 0
	testUserWSRoster.userIDMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(1, mapLen, "expected two entries in userIDMap")

	// Check that remove of non-existing user does not panic and move other users.
	testUserWSRoster.RemoveID("testing123")

	mapLen = 0
	testUserWSRoster.userIDMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(1, mapLen, "expected two entries in userIDMap")

	// Check that existing user gets deleted.
	testUserWSRoster.RemoveID(id)

	mapLen = 0
	testUserWSRoster.userIDMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(0, mapLen, "expected two entries in userIDMap")
}
