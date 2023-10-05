package services

import (
	"context"
)

type UserServicer interface {
	ReadLoop(context.Context)
	WriteLoop(context.Context)
	GetWriteChan() chan []byte
}
