package main

import (
	"fmt"
	"github.com/google/shlex"
	"strings"
)

type CommandTarget string

const (
	CommandTargetClient CommandTarget = "client"
	CommandTargetRoom   CommandTarget = "room"
)

type OPLevel uint8

const (
	OPLevelNone OPLevel = iota
	OPLevelUser
	OPLevelAdmin
)

func ParseOPLevel(level string) (OPLevel, error) {
	switch level {
	case "none":
		return OPLevelNone, nil
	case "user":
		return OPLevelUser, nil
	case "admin":
		return OPLevelAdmin, nil
	default:
		return OPLevelNone, fmt.Errorf("unknown OP level: %s", level)
	}
}

func (l OPLevel) String() string {
	switch l {
	case OPLevelNone:
		return "none"
	case OPLevelUser:
		return "user"
	case OPLevelAdmin:
		return "admin"
	default:
		return "unknown"
	}
}

type CommandSpec struct {
	Name             string
	Desc             string
	Help             string
	ArgsMin, ArgsMax int
	Target           CommandTarget
	OPLevel          OPLevel
}

type Command struct {
	Name    string
	Args    []string
	Target  CommandTarget
	OPLevel OPLevel
}

var Commands = map[string]CommandSpec{
	"join": {
		Name:    "join",
		Desc:    "join a room",
		Help:    "/join <room> [password]",
		ArgsMin: 1,
		ArgsMax: 2,
		Target:  CommandTargetClient,
		OPLevel: OPLevelNone,
	},
	"start": {
		Name:    "start",
		Desc:    "start a new room",
		Help:    "/start <room> [password]",
		ArgsMin: 1,
		ArgsMax: 2,
		Target:  CommandTargetClient,
		OPLevel: OPLevelNone,
	},
	"exit": {
		Name:    "exit",
		Desc:    "exit the current room",
		Help:    "/exit",
		ArgsMin: 0,
		ArgsMax: 0,
		Target:  CommandTargetClient,
		OPLevel: OPLevelNone,
	},
	"nick": {
		Name:    "nick",
		Desc:    "change or set your nickname",
		Help:    "/nick [nick]",
		ArgsMin: 1,
		ArgsMax: 1,
		Target:  CommandTargetClient,
		OPLevel: OPLevelUser,
	},
	"who": {
		Name:    "who",
		Desc:    "list all users in the current room",
		Help:    "/who",
		ArgsMin: 0,
		ArgsMax: 0,
		Target:  CommandTargetRoom,
		OPLevel: OPLevelUser,
	},
	"w": {
		Name:    "w",
		Desc:    "send a direct message to a user",
		Help:    "/w <nickname> <message>",
		ArgsMin: 2,
		ArgsMax: 2,
		Target:  CommandTargetRoom,
		OPLevel: OPLevelUser,
	},
	"clear": {
		Name:    "clear",
		Desc:    "clear the chat window",
		Help:    "/clear",
		ArgsMin: 0,
		ArgsMax: 0,
		OPLevel: OPLevelNone,
	},
	"help": {
		Name:    "help",
		Desc:    "list all commands, or get help for a specific command",
		Help:    "/help [command]",
		ArgsMin: 0,
		ArgsMax: 1,
		Target:  CommandTargetClient,
		OPLevel: OPLevelNone,
	},
	"op": {
		Name:    "op",
		Desc:    "change permission levels",
		Help:    "/op <user> <level>",
		ArgsMin: 2,
		ArgsMax: 2,
		Target:  CommandTargetRoom,
		OPLevel: OPLevelAdmin,
	},
	"welcome": {
		Name:    "welcome",
		Desc:    "set or clear the welcome message",
		Help:    "/welcome [message]",
		ArgsMin: 0,
		ArgsMax: 1,
		Target:  CommandTargetRoom,
		OPLevel: OPLevelAdmin,
	},
	"password": {
		Name:    "password",
		Desc:    "set or clear the password for the room",
		Help:    "/password [password]",
		ArgsMin: 0,
		ArgsMax: 1,
		Target:  CommandTargetRoom,
		OPLevel: OPLevelAdmin,
	},
}

func ParseCommand(input string) (*Command, error) {
	input = strings.TrimSpace(input)
	if input == "" || !strings.HasPrefix(input, "/") {
		return nil, nil
	}

	tokens, err := shlex.Split(input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	name := strings.TrimPrefix(tokens[0], "/")
	args := tokens[1:]
	command := &Command{
		Name: name,
		Args: args,
	}

	spec, ok := Commands[name]
	if !ok {
		return command, fmt.Errorf("unknown command: %s", name)
	}

	if len(args) < spec.ArgsMin {
		return command, fmt.Errorf("missing arguments for: %s\nUsage:\n  %s", name, spec.Help)
	}

	if len(args) > spec.ArgsMax {
		return command, fmt.Errorf("too many arguments for: %s\nUsage:\n  %s", name, spec.Help)
	}

	return &Command{
		Name:    name,
		Args:    args,
		Target:  spec.Target,
		OPLevel: spec.OPLevel,
	}, nil
}

func Help(args []string) string {
	if len(args) == 0 {
		commands := make([]string, 0, len(Commands))
		for _, command := range Commands {
			commands = append(commands, fmt.Sprintf("/%s: %s", command.Name, command.Desc))
		}
		return strings.Join(commands, "\n")
	} else {
		command, ok := Commands[args[0]]
		if !ok {
			return "unknown command"
		}
		return fmt.Sprintf("/%s: %s\n  %s", command.Name, command.Desc, command.Help)
	}
}
