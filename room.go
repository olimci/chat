package main

import (
	"errors"
	"fmt"
	"strings"
)

var (
	roomErrShouldQuit = errors.New("room should quit")
)

type Room struct {
	Name string

	Clients map[*Client]ClientDataExternal

	External chan ClientMessage // External broadcast channel
	Internal chan RoomMessage   // Internal broadcast channel

	Register   chan RegisterRequest
	Unregister chan UnregisterRequest

	Rules *Rules
}

func NewRoom(name string) *Room {
	return &Room{
		Name: name,

		Clients: make(map[*Client]ClientDataExternal),

		External: make(chan ClientMessage, 512),
		Internal: make(chan RoomMessage, 512),

		Register:   make(chan RegisterRequest, 256),
		Unregister: make(chan UnregisterRequest, 256),

		Rules: NewRules(),
	}
}

func (r *Room) Run() {
	for {
		select {
		case req := <-r.Register:
			data := NewClientDataExternal(req.Creator)

			r.Clients[req.Client] = data

			if !r.Rules.noCommands && req.WantsNick != "" {
				r.setNick(req.Client, req.WantsNick)
			}

			if r.Rules.hasWelcomeMessage {
				req.Client.Send <- RoomMessage{
					Type: MessageTypeNotice,
					Body: r.Rules.welcomeMessage,
				}.Fill()
			}

		case req := <-r.Unregister:
			data, ok := r.Clients[req.Client]
			if !ok {
				continue
			}

			if data.Nick != "" {
				r.Internal <- RoomMessage{
					Type: MessageTypeLeave,
					Body: fmt.Sprintf("%s left the room: %s", data.Nick, req.Reason),
					Target: Target{
						Type: TargetTypeAll,
					},
				}.Fill()
			}

			err := r.remove(req.Client)
			if r.shouldQuit(err) {
				return
			}

		case message := <-r.Internal:
			err := r.handleInternal(message)
			if r.shouldQuit(err) {
				return
			}

		case message := <-r.External:
			if r.Rules.noMessages {
				r.Internal <- RoomMessage{
					Type: MessageTypeError,
					Body: "messages are disabled on this room",
					Target: Target{
						Type:   TargetTypeOne,
						Client: message.Client,
					},
				}.Fill()
				continue
			}

			err := r.handleExternal(message)
			if r.shouldQuit(err) {
				return
			}
		}
	}
}

func (r *Room) shouldQuit(err error) bool {
	if r.Rules.keepOpen {
		return false
	}

	return errors.Is(err, roomErrShouldQuit)
}

func (r *Room) remove(client *Client) error {
	delete(r.Clients, client)
	if !r.Rules.keepOpen && len(r.Clients) == 0 {
		Rooms.Delete(r.Name)
		return roomErrShouldQuit
	}

	return nil
}

func (r *Room) send(client *Client, message RoomMessage) error {
	select {
	case client.Send <- message:
		return nil
	default:
		return r.remove(client)
	}
}

func (r *Room) handleInternal(message RoomMessage) error {
	for client, data := range r.Clients {
		if !message.Target.Should(client, data) {
			continue
		}

		err := r.send(client, message)
		if errors.Is(err, roomErrShouldQuit) {
			return roomErrShouldQuit
		}
	}

	return nil
}

func (r *Room) handleExternal(message ClientMessage) error {
	switch message.Type {
	case MessageTypeMessage:
		data := r.Clients[message.Client]

		if data.Nick == "" {
			r.Internal <- RoomMessage{
				Type: MessageTypeError,
				Body: "you must set a nickname with /nick before sending messages",
				Target: Target{
					Type:   TargetTypeOne,
					Client: message.Client,
				},
			}.Fill()
			return nil
		}

		err := r.handleInternal(message.Promote(data))
		if errors.Is(err, roomErrShouldQuit) {
			return roomErrShouldQuit
		}

		return nil

	case MessageTypeCommand:
		if message.Command == nil {
			return nil
		}

		if r.Rules.noCommands {
			r.Internal <- RoomMessage{
				Type: MessageTypeError,
				Body: "commands are disabled on this room",
				Target: Target{
					Type:   TargetTypeOne,
					Client: message.Client,
				},
			}.Fill()
		}

		command := message.Command

		data := r.Clients[message.Client]

		if command.OPLevel > data.OPLevel {
			r.Internal <- RoomMessage{
				Type: MessageTypeError,
				Body: fmt.Sprintf("insufficient permission (%s) to use %s (%s)", data.OPLevel, command.Name, command.OPLevel),
				Target: Target{
					Type:   TargetTypeOne,
					Client: message.Client,
				},
			}.Fill()
			return nil
		}

		switch command.Name {
		case "nick":
			r.setNick(message.Client, command.Args[0])
		case "who":
			r.getWho(message.Client)
		case "w":
			r.whisper(message.Client, command.Args[0], command.Args[1])
		case "op":
			r.op(message.Client, command.Args[0], command.Args[1])
		case "welcome":
			var welcomeMessage *string
			if len(command.Args) > 0 {
				welcomeMessage = &command.Args[0]
			}

			r.welcome(message.Client, welcomeMessage)
		case "password":
			var password *string
			if len(command.Args) > 0 {
				password = &command.Args[0]
			}

			r.password(message.Client, password)
		}

		return nil

	default:
		return nil
	}
}

func (r *Room) setNick(client *Client, nick string) {
	newNick := strings.TrimSpace(nick)
	if newNick == "" {
		r.Internal <- RoomMessage{
			Type: MessageTypeError,
			Body: "nickname cannot be empty",
			Target: Target{
				Type:   TargetTypeOne,
				Client: client,
			},
		}.Fill()
		return
	}

	for other, data := range r.Clients {
		if other != client && data.Nick == newNick {
			r.Internal <- RoomMessage{
				Type: MessageTypeError,
				Body: fmt.Sprintf("nickname %s is already in use", newNick),
				Target: Target{
					Type:   TargetTypeOne,
					Client: client,
				},
			}.Fill()
			return
		}
	}

	data := r.Clients[client]
	oldNick := data.Nick
	data.Nick = newNick
	r.Clients[client] = data

	if oldNick != "" {
		r.Internal <- RoomMessage{
			Type: MessageTypeNotice,
			Body: fmt.Sprintf("%s changed their nickname to %s", oldNick, newNick),
		}.Fill()
	} else {
		r.Internal <- RoomMessage{
			Type: MessageTypeJoin,
			Body: fmt.Sprintf("%s joined the room", newNick),
		}.Fill()
	}
}

func (r *Room) getWho(client *Client) {
	online := make([]string, 0, len(r.Clients))

	for client, data := range r.Clients {
		if data.Nick == "" {
			continue
		}

		online = append(online, fmt.Sprintf("%s (%s)", data.Nick, client.Conn.RemoteAddr().String()))
	}

	var body string
	if len(online) == 0 {
		body = "no one is currently online"
	} else {
		body = fmt.Sprintf("currently online:\n%s", strings.Join(online, "\n"))
	}

	r.Internal <- RoomMessage{
		Type: MessageTypeCommand,
		Body: body,
		Target: Target{
			Type:   TargetTypeOne,
			Client: client,
		},
	}.Fill()
}

func (r *Room) whisper(client *Client, nick string, message string) {
	data, ok := r.Clients[client]
	if !ok {
		return
	}

	if data.Nick == "" {
		r.Internal <- RoomMessage{
			Type: MessageTypeError,
			Body: "you must set a nickname before sending messages",
			Target: Target{
				Type:   TargetTypeOne,
				Client: client,
			},
		}.Fill()
		return
	}

	r.Internal <- RoomMessage{
		Type:  MessageTypeWhisper,
		Nick:  data.Nick,
		Color: data.Color,
		Body:  fmt.Sprintf("whispers: %s", message),
		Target: Target{
			Type: TargetTypeNickOne,
			Nick: nick,
		},
	}.Fill()

	r.Internal <- RoomMessage{
		Type:  MessageTypeWhisper,
		Nick:  data.Nick,
		Color: data.Color,
		Body:  fmt.Sprintf("whispers: %s", message),
		Target: Target{
			Type:   TargetTypeOne,
			Client: client,
		},
	}.Fill()
}

func (r *Room) op(client *Client, nick string, levelName string) {
	level, err := ParseOPLevel(levelName)
	if err != nil {
		r.Internal <- RoomMessage{
			Type: MessageTypeError,
			Body: err.Error(),
			Target: Target{
				Type:   TargetTypeOne,
				Client: client,
			},
		}.Fill()
		return
	}

	var target *Client
	for test, data := range r.Clients {
		if data.Nick == nick {
			target = test
			break
		}
	}

	if target == nil {
		r.Internal <- RoomMessage{
			Type: MessageTypeError,
			Body: fmt.Sprintf("user %s is not online", nick),
			Target: Target{
				Type:   TargetTypeOne,
				Client: client,
			},
		}.Fill()
		return
	}

	data := r.Clients[client]
	data.OPLevel = level
	r.Clients[client] = data

	r.Internal <- RoomMessage{
		Type: MessageTypeNotice,
		Body: fmt.Sprintf("%s's permission level is now %s", nick, level),
		Target: Target{
			Type:   TargetTypeOne,
			Client: client,
		},
	}.Fill()

	r.Internal <- RoomMessage{
		Type: MessageTypeNotice,
		Body: fmt.Sprintf("your permission level is now %s", level),
		Target: Target{
			Type:   TargetTypeOne,
			Client: target,
		},
	}.Fill()
}

func (r *Room) welcome(client *Client, message *string) {
	if *message == "" {
		r.Rules.hasWelcomeMessage = false
		r.Internal <- RoomMessage{
			Type: MessageTypeNotice,
			Body: "welcome message disabled",
			Target: Target{
				Type:   TargetTypeOne,
				Client: client,
			},
		}.Fill()
	} else {
		r.Rules.hasWelcomeMessage = true
		r.Rules.welcomeMessage = *message
		r.Internal <- RoomMessage{
			Type: MessageTypeNotice,
			Body: fmt.Sprintf("welcome message set to: %s", *message),
			Target: Target{
				Type:   TargetTypeOne,
				Client: client,
			},
		}.Fill()
	}
}

func (r *Room) password(client *Client, password *string) {
	if password == nil {
		r.Rules.hasPassword = false
		r.Internal <- RoomMessage{
			Type: MessageTypeNotice,
			Body: "password disabled",
			Target: Target{
				Type:   TargetTypeOne,
				Client: client,
			},
		}.Fill()
	} else {
		r.Rules.hasPassword = true
		r.Rules.password = *password
		r.Internal <- RoomMessage{
			Type: MessageTypeNotice,
			Body: fmt.Sprintf("password set to: %s", *password),
			Target: Target{
				Type:   TargetTypeOne,
				Client: client,
			},
		}.Fill()
	}
}
