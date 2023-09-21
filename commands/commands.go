package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/items"
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
	case GET:
		return RunGetCommand(pc.Arguments, ch, loc)
	case DROP:
		return RunDropCommand(pc.Arguments, ch, loc)
	case INVENTORY:
		return RunInvCommand(ch)
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
		contents := chLoc.ListContents()
		contStrings := ""
		for _, v := range contents {
			contStrings += v + "\n"
		}
		resp = fmt.Sprintf("%v\n    %v\nExits: %v\n", chLoc.Name, chLoc.Desc, chLoc.Exits)
		if contStrings != "" {
			resp += contStrings
		}
		resp += "\n"
	case ME:
		resp = ch.CharData.Desc + "\n\n"
	case IDENT:
		if itm, err := items.AutoCompleteItems(args[0].Literal, ch.CharData.Inv); err == nil {
			resp = itm.Desc + "\n\n"
		} else if itm, err := items.AutoCompleteItems(args[0].Literal, chLoc.Contents); err == nil {
			resp = itm.Desc + "\n\n"
		} else {
			resp = fmt.Sprintf("You don't see %v here.\n\n", args[0].Literal)
		}
	default:
		resp = fmt.Sprintf("You don't see %v here.\n\n", args[0].Literal)
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

func RunGetCommand(args []Token, ch *chara.ActiveCharacter, loc rooms.RoomList) error {
	chLoc := loc[ch.CharData.Location]
	if len(args) == 0 {
		ch.ResponseChannel <- "Get what?\n\n"
		return nil
	} else {
		itm, err := items.AutoCompleteItems(args[0].Literal, chLoc.Contents)
		if err != nil {
			ch.ResponseChannel <- fmt.Sprintf("You don't see %q here.\n\n", args[0].Literal)
			return err
		} else {
			chLoc.Remove(itm.Serial)
			ch.CharData.Insert(itm)
			ch.ResponseChannel <- fmt.Sprintf("You pick up the %v.\n\n", itm.Name)
			return nil
		}
	}
}

func RunDropCommand(args []Token, ch *chara.ActiveCharacter, loc rooms.RoomList) error {
	chLoc := loc[ch.CharData.Location]
	if len(args) == 0 {
		ch.ResponseChannel <- "Get what?\n\n"
		return nil
	} else {
		itm, err := items.AutoCompleteItems(args[0].Literal, ch.CharData.Inv)
		if err != nil {
			ch.ResponseChannel <- fmt.Sprintf("You don't have a %q.\n\n", args[0].Literal)
			return err
		} else {
			ch.CharData.Remove(itm.Serial)
			chLoc.Insert(itm)
			ch.ResponseChannel <- fmt.Sprintf("You drop the %v on the ground.\n\n", itm.Name)
			return nil
		}
	}
}

func RunInvCommand(ch *chara.ActiveCharacter) error {
	chInv := ch.CharData.ListContents()
	resp := ""
	if len(chInv) > 0 {
		resp = "You are carrying:\n"
		for _, v := range chInv {
			resp += v + "\n"
		}
		resp += "\n"
	} else {
		resp = "You are not carrying anything.\n\n"
	}
	ch.ResponseChannel <- resp
	return nil
}
