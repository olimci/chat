package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"strings"
	"sync"
)

type RegisterRequest struct {
	Client    *Client
	WantsNick string
	Creator   bool
}

type UnregisterRequest struct {
	Client *Client
	Reason string
}

type Client struct {
	Conn *websocket.Conn

	Send chan RoomMessage
	Recv chan ClientMessage

	Room *Room

	Data ClientDataInternal
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		Conn: conn,
		Send: make(chan RoomMessage, 256),
		Recv: make(chan ClientMessage, 256),
	}
}

func (c *Client) readPump() {
	defer func() {
		_ = c.Conn.Close()
	}()

	for {
		var frame WSFrame
		if err := c.Conn.ReadJSON(&frame); err != nil {
			break
		}

		if strings.TrimSpace(frame.Message) == "" {
			continue
		}

		c.Recv <- ClientMessage{
			Type:   MessageTypeMessage,
			Client: c,
			Body:   frame.Message,
		}
	}
}

func (c *Client) writePump() {
	for msg := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Render()))
		if err != nil {
			break
		}
	}
}

func (c *Client) start(roomName string, password *string) {
	// Check the target room doesn't exist
	_, ok := Rooms.Get(roomName)
	if ok {
		c.Send <- RoomMessage{
			Type: MessageTypeError,
			Body: fmt.Sprintf("room %s already exists", roomName),
		}.Fill()
		return
	}

	// Exit current room
	if c.Room != nil {
		c.Room.Unregister <- UnregisterRequest{
			Client: c,
			Reason: fmt.Sprintf("starting room %s", roomName),
		}

		flush(c.Send)
		flush(c.Recv)

		c.Room = nil

		c.Send <- RoomMessage{
			Type: MessageTypeReset,
		}
	}

	// Create the room
	room := NewRoom(roomName)
	if password != nil {
		room.Rules.Password(*password)
	}
	Rooms.Set(roomName, room)
	go room.Run()

	// Join
	c.Room = room
	room.Register <- RegisterRequest{
		Client:    c,
		WantsNick: c.Data.Nick,
		Creator:   true,
	}
}

func (c *Client) join(roomName string, password *string) {
	room, ok := Rooms.Get(roomName)
	if !ok {
		c.Send <- RoomMessage{
			Type: MessageTypeError,
			Body: fmt.Sprintf("room %s does not exist", roomName),
		}.Fill()
		return
	}

	// Check password
	if room.Rules.hasPassword {
		if password == nil {
			c.Send <- RoomMessage{
				Type: MessageTypeError,
				Body: "room has a password",
			}.Fill()
			return
		} else if room.Rules.password != *password {
			c.Send <- RoomMessage{
				Type: MessageTypeError,
				Body: "incorrect password",
			}.Fill()
			return
		}
	}

	// Exit current room
	if c.Room != nil {
		c.Room.Unregister <- UnregisterRequest{
			Client: c,
			Reason: fmt.Sprintf("joining room %s", roomName),
		}

		flush(c.Send)
		flush(c.Recv)

		c.Room = nil

		c.Send <- RoomMessage{
			Type: MessageTypeReset,
		}
	}

	// Join room
	c.Room = room
	room.Register <- RegisterRequest{
		Client:    c,
		WantsNick: c.Data.Nick,
	}
}

func (c *Client) command(command *Command) {
	switch command.Name {
	case "join":
		var password *string
		if len(command.Args) > 1 {
			password = &command.Args[1]
		}

		c.join(command.Args[0], password)

	case "start":
		var password *string
		if len(command.Args) > 1 {
			password = &command.Args[1]
		}

		c.start(command.Args[0], password)
	case "exit":
		c.join("main", nil)

	case "clear":
		c.Send <- RoomMessage{
			Type: MessageTypeReset,
		}

	case "nick":
		c.Data.Nick = command.Args[0]
		if !c.Room.Rules.noCommands {
			c.Room.External <- ClientMessage{
				Type:    MessageTypeCommand,
				Client:  c,
				Command: command,
			}
		}

	case "help":
		c.Send <- RoomMessage{
			Type: MessageTypeCommand,
			Body: Help(command.Args),
		}.Fill()
	}
}

func (c *Client) handle(done chan struct{}) {
	c.Send <- RoomMessage{
		Type: MessageTypeReset,
	}

	for {
		select {
		case <-done:
			return
		case msg := <-c.Recv:
			command, err := ParseCommand(msg.Body)
			if command == nil {
				c.Room.External <- msg
				continue
			}

			if err != nil {
				c.Send <- RoomMessage{
					Type: MessageTypeError,
					Body: err.Error(),
				}.Fill()
				continue
			}

			if command.Target == CommandTargetRoom {
				c.Room.External <- ClientMessage{
					Type:    MessageTypeCommand,
					Client:  c,
					Command: command,
				}
			} else {
				c.command(command)
			}
		}
	}
}

func (c *Client) Serve() {
	done := make(chan struct{})
	var once sync.Once

	c.join(DefaultRoom, nil)

	closer := func() {
		once.Do(func() {
			if c.Room != nil {
				c.Room.Unregister <- UnregisterRequest{
					Client: c,
					Reason: "quit",
				}
			}
			close(c.Send)
			close(done)
		})
	}

	go func() {
		c.readPump()
		closer()
	}()

	go func() {
		c.writePump()
		closer()
	}()

	c.handle(done)
}
