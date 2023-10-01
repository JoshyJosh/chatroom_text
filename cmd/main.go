package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/exp/slog"
)

func main() {
	httpPort := fmt.Sprintf(":%s", *(flag.String("port", "8080", "Listen address")))

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil).WithAttrs(
		[]slog.Attr{
			{
				Key:   "stage",
				Value: slog.StringValue("main"),
			},
		},
	))
	logger.Info("starting")
	defer logger.Info("stopping")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../static"))))
	http.HandleFunc("/", enterChat)

	log.Fatal(http.ListenAndServeTLS(httpPort, "cert.pem", "key.pem", nil))
}

func enterChat(w http.ResponseWriter, r *http.Request) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil).WithAttrs(
		[]slog.Attr{
			{
				Key:   "stage",
				Value: slog.StringValue("enterChat"),
			},
		},
	))

	if pusher, ok := w.(http.Pusher); ok {
		logger.Info("pushed http2")
		// Push is supported.
		options := &http.PushOptions{
			Header: http.Header{
				"Accept-Encoding": r.Header["Accept-Encoding"],
			},
		}
		if err := pusher.Push("/static/index.js", options); err != nil {
			logger.Error("Failed to push: %w", err)
		}
	}

	http.ServeFile(w, r, "static/index.html")
}

func sendMessagefunc(w http.ResponseWriter, r *http.Request) {
	// @todo make send message func
}
