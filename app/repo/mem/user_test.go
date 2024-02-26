package mem

import (
	"chatroom_text/models"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserWSRosterAddUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	testUserWSRoster := userWSRoster{
		userIDMap: &sync.Map{},
	}

	user1 := models.User{
		ID: uuid.New(),
	}
	err := testUserWSRoster.AddUser(user1)
	if err != nil {
		t.Fatal(err)
	}

	// make sure user1 ID is not same as user2
	user2ID := uuid.New()
	for user2ID == user1.ID {
		user2ID = uuid.New()
	}

	user2 := models.User{
		ID: user2ID,
	}

	err = testUserWSRoster.AddUser(user2)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(user1, user2, "id's should be unique")

	mapLen := 0
	testUserWSRoster.userIDMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(2, mapLen, "expected two entries in userIDMap")
}

func TestUserWSRosterAddUserError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	testUserWSRoster := userWSRoster{
		userIDMap: &sync.Map{},
	}

	user1 := models.User{}
	err := testUserWSRoster.AddUser(user1)
	if err != nil {
		t.Fatal(err)
	}

	user2 := models.User{}
	err = testUserWSRoster.AddUser(user2)
	assert.Error(err)
}

func TestUserWSRosterRemoveUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	testUserWSRoster := userWSRoster{
		userIDMap: &sync.Map{},
	}

	user := models.User{
		ID: uuid.New(),
	}

	if err := testUserWSRoster.AddUser(user); err != nil {
		t.Fatal(err)
	}

	mapLen := 0
	testUserWSRoster.userIDMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(1, mapLen, "expected one entry in userIDMap")

	// Check that remove of non-existing user does not panic and move other users.
	testUserWSRoster.RemoveID(uuid.UUID{})

	mapLen = 0
	testUserWSRoster.userIDMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(1, mapLen, "expected one entry in userIDMap")

	// Check that existing user gets deleted.
	testUserWSRoster.RemoveID(user.ID)

	mapLen = 0
	testUserWSRoster.userIDMap.Range(func(_, _ any) bool {
		mapLen++
		return true
	})

	assert.Equal(0, mapLen, "expected zero entries in userIDMap")
}
