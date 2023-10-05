package services

type ChatroomServicer interface {
	AddUser(id string, user UserServicer) error
	AddNewUser(user UserServicer) string
}
