package models

import (
	"sync"
	"time"
)

type userMap struct {
	Map   map[string]*User
	mutex sync.Mutex
}

type messageList struct {
	list  []*Message
	mutex sync.Mutex
}

type Chatroom struct {
	Name        string
	UserMap     userMap
	MessageList messageList
}

type User struct {
	Name string
}

type Message struct {
	timestamp time.Time
	body      string
	author    *User
}
