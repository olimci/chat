package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

var templates = template.Must(template.ParseGlob("templates/*.tmpl"))

const DefaultRoom = "main"

var Rooms = NewMuMap[string, *Room]()

func main() {
	// Initialise router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Serve static files
	workDir, _ := filepath.Abs(".")
	staticDir := http.Dir(filepath.Join(workDir, "static"))
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticDir)))

	r.Get("/", Handler)
	r.Get("/ws", WSHandler)
	// r.Get("/wsapi", WSAPIHandler)

	roomMain := NewRoom(DefaultRoom)
	roomMain.Rules.
		KeepOpen().
		NoCommands().
		NoMessages().
		WelcomeMessage("welcome to e74chat.\n - messages are disabled in the main lobby\n - please use /start or /join to start chatting\n - you can run /help for a list of commands")

	Rooms.Set(DefaultRoom, roomMain)
	go roomMain.Run()

	// Start server
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func Handler(w http.ResponseWriter, _ *http.Request) {
	execute(w, "room.tmpl", nil)
}
