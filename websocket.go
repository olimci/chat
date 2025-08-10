package main

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type WSFrame struct {
	Message string `json:"message"`
}

// Configure the upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for now
	// Probably best bet is to add an /wsapi for external origins
	// and then build out a separate json-only client type.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	NewClient(conn).Serve()
}
