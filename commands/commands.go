package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/rooms"
)

type ParsedCommand struct {
	Command   Token
	Arguments []Token
}

func ParseCommand(in string) (*ParsedCommand, error) {
	words := strings.Split(strings.ToLower(in), " ")
	if len(words) == 0 {
		return nil, errors.New("empty command")
	}
	newCmd := lookupCommand(words[0])
	newArgs := []Token{}
	for _, v := range words[1:] {
		newArgs = append(newArgs, lookupIdent(v))
	}
	return &ParsedCommand{newCmd, newArgs}, nil
}

func RunCommand(pc *ParsedCommand, ch *chara.ActiveCharacter, loc rooms.RoomList) error {
	switch pc.Command.Type {
	case LOOK:
		return RunLookCommand(pc.Arguments, ch, loc)
	case GO:
		return RunGoCommand(pc.Arguments, ch, loc)
	case DIRECTION:
		args := []Token{{IDENT, pc.Command.Literal}}
		return RunGoCommand(args, ch, loc)
	default:
		return fmt.Errorf("command %q not handled", pc.Command.Literal)
	}
}

func RunLookCommand(args []Token, ch *chara.ActiveCharacter, loc rooms.RoomList) error {
	resp := ""
	chLoc := loc[ch.CharData.Location]
	if len(args) == 0 {
		args = append(args, Token{HERE, "here"})
	}
	switch args[0].Type {
	case HERE:
		resp = fmt.Sprintf("%v\n    %v\nExits: %v\n\n", chLoc.Name, chLoc.Desc, chLoc.Exits)
	case ME:
		resp = ch.CharData.Desc + "\n\n"
	default:
		resp = "You don't see %v here.\n\n"
	}
	ch.ResponseChannel <- resp
	return nil
}

func RunGoCommand(args []Token, ch *chara.ActiveCharacter, loc rooms.RoomList) error {
	chLoc := loc[ch.CharData.Location]
	if len(args) == 0 {
		ch.ResponseChannel <- "Go where?\n\n"
	} else {
		destString := AutoCompleteDirs(args[0].Literal)
		if dest, ok := chLoc.Exits[destString]; ok {
			ch.CharData.Location = dest
			ch.ResponseChannel <- fmt.Sprintf("You walk %v.\n\n", destString)
			RunLookCommand([]Token{}, ch, loc)
		} else {
			ch.ResponseChannel <- "You can't go that way.\n\n"
		}
	}
	return nil
}
