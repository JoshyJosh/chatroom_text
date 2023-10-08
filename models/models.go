package models

import (
	"time"
)

type WSMessage struct {
	Text      string    `json:"msg"`
	Timestamp time.Time `json:"timestamp"`
	ClientID  string    `json:"clientID"` // should corelate with WSClient ID
}

type UserWS struct {
	ID        string
	WriteChan chan []byte
}
