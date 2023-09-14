package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("staring")
	defer fmt.Println("stopping")
}

func enterChat(func(w http.ResponseWriter, r *http.Request)) {
	// @todo make entry to chatroom
}

func sendMessagefunc(w http.ResponseWriter, r *http.Request) {
	// @todo make send message func
}
