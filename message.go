package main

import (
	"fmt"
	"strings"
	"time"
)

type MessageType string

const (
	MessageTypeMessage MessageType = "message"
	MessageTypeCommand MessageType = "command"
	MessageTypeWhisper MessageType = "whisper"
	MessageTypeError   MessageType = "error"
	MessageTypeNotice  MessageType = "notice"
	MessageTypeJoin    MessageType = "join"
	MessageTypeLeave   MessageType = "leave"
	MessageTypeReset   MessageType = "reset"
)

type ClientMessage struct {
	Type    MessageType
	Client  *Client
	Body    string
	Command *Command
}

func (m ClientMessage) Promote(data ClientDataExternal) RoomMessage {
	return RoomMessage{
		ID:    messageID(),
		Type:  MessageTypeMessage,
		Time:  time.Now(),
		Nick:  data.Nick,
		Color: data.Color,
		Body:  m.Body,
		Target: Target{
			Type: TargetTypeAll,
		},
	}
}

var messageColors = map[MessageType]string{
	MessageTypeMessage: "#cdd6f4", // text – calm lavender-ice foreground
	MessageTypeCommand: "#74c7ec", // sapphire – bright enough to pop for “/cmd”
	MessageTypeWhisper: "#cba6f7", // mauve – subtle, gentle lavender for whispers
	MessageTypeError:   "#f38ba8", // red – unmistakable but still pastel
	MessageTypeNotice:  "#f9e2af", // yellow – gentle alert / info
	MessageTypeJoin:    "#a6e3a1", // green – success/positive event
	MessageTypeLeave:   "#eba0ac", // maroon – softer farewell than pure red
}

type RoomMessage struct {
	ID     string      `json:"id"`
	Type   MessageType `json:"type"`
	Time   time.Time   `json:"time"`
	Nick   string      `json:"nick"`
	Color  string      `json:"color"`
	Body   string      `json:"body"`
	Target Target      `json:"target"`
}

func (m RoomMessage) Fill() RoomMessage {
	if m.ID == "" {
		m.ID = messageID()
	}

	if m.Time.IsZero() {
		m.Time = time.Now()
	}

	if m.Nick == "" {
		m.Nick = "*"
	}

	if m.Color == "" {
		m.Color = messageColors[m.Type]
	}

	if m.Target.Type == "" {
		m.Target.Type = TargetTypeAll
	}

	return m
}

func (m RoomMessage) Render() string {
	var buf strings.Builder
	err := templates.ExecuteTemplate(&buf, "message.tmpl", m)
	if err != nil {
		return "failed to render message"
	}
	return strings.TrimSpace(buf.String())
}

func messageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

type TargetType string

const (
	TargetTypeAll        TargetType = "room"
	TargetTypeNickOne    TargetType = "nick_user"
	TargetTypeNickOthers TargetType = "nick_others"
	TargetTypeOne        TargetType = "user"
	TargetTypeOthers     TargetType = "others"
)

type Target struct {
	Type   TargetType
	Nick   string
	Client *Client
}

func (t Target) Should(client *Client, clientData ClientDataExternal) bool {
	switch t.Type {
	case TargetTypeAll:
		return true
	case TargetTypeNickOne:
		return t.Nick == clientData.Nick
	case TargetTypeNickOthers:
		return t.Nick != clientData.Nick
	case TargetTypeOne:
		return client == t.Client
	case TargetTypeOthers:
		return client != t.Client
	default:
		return false
	}
}
