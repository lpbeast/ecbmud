package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/items"
	"github.com/lpbeast/ecbmud/mobs"
	"github.com/lpbeast/ecbmud/rooms"
)

type ParsedCommand struct {
	Command   Token
	Arguments string
}

func ParseCommand(in string) (*ParsedCommand, error) {
	cmd, args, _ := strings.Cut(in, " ")
	if cmd == "" {
		return nil, errors.New("empty command")
	}
	newCmd := lookupCommand(cmd)
	return &ParsedCommand{newCmd, args}, nil
}

// TODO: expand for parsing quantities and ordinals
func ParseArgs(in string) []Token {
	// strings.Split seems to do something weird with an empty string
	if len(in) == 0 {
		return []Token{}
	}
	words := strings.Split(strings.ToLower(in), " ")
	newArgs := []Token{}
	for _, v := range words {
		newArgs = append(newArgs, lookupIdent(v))
	}
	return newArgs
}

func RunCommand(pc *ParsedCommand, ch *chara.ActiveCharacter, loc rooms.RoomList, charas chara.UserList) error {
	switch pc.Command.Type {
	case LOOK:
		return RunLookCommand(ParseArgs(pc.Arguments), ch, loc)
	case GO:
		return RunGoCommand(ParseArgs(pc.Arguments), ch, loc)
	case GET:
		return RunGetCommand(ParseArgs(pc.Arguments), ch, loc)
	case DROP:
		return RunDropCommand(ParseArgs(pc.Arguments), ch, loc)
	case INVENTORY:
		return RunInvCommand(ch)
	case DIRECTION:
		args := []Token{{IDENT, pc.Command.Literal}}
		return RunGoCommand(args, ch, loc)
	case SAY:
		return RunSayCommand(pc.Arguments, ch, loc)
	case TELL:
		return RunTellCommand(pc.Arguments, ch, charas)
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
		pcAndMobStrings := ""
		for _, v := range chLoc.PCs {
			if v != ch {
				pcAndMobStrings += v.CharData.Name + " is standing here.\n"
			}
		}
		for _, v := range chLoc.Mobs {
			pcAndMobStrings += v.Name + " is standing here.\n"
		}
		contents := chLoc.ListContents()
		contStrings := ""
		for _, v := range contents {
			contStrings += v + "\n"
		}
		resp = fmt.Sprintf("%v\n    %v\nExits: %v\n", chLoc.Name, chLoc.Desc, chLoc.Exits)
		if pcAndMobStrings != "" {
			resp += pcAndMobStrings
		}
		if contStrings != "" {
			resp += contStrings
		}
	case ME:
		resp = ch.CharData.Desc + "\n"
	case IDENT:
		fmt.Printf("arguments: %+v\n", args)
		if itm, err := items.AutoCompleteItems(args[0].Literal, ch.CharData.Inv); err == nil {
			resp = fmt.Sprintf("%s\n", itm.Desc)
		} else if itm, err := items.AutoCompleteItems(args[0].Literal, chLoc.Contents); err == nil {
			resp = fmt.Sprintf("%s\n", itm.Desc)
		} else if ch, err := chara.AutoCompletePCs(args[0].Literal, chLoc.PCs); err == nil {
			resp = fmt.Sprintf("You look at %s.\n%s\n", ch.CharData.Name, ch.CharData.Desc)
		} else if m, err := mobs.AutoCompleteMobs(args[0].Literal, chLoc.Mobs); err == nil {
			resp = fmt.Sprintf("You look at %s.\n%s\n", m.Name, m.Desc)
		} else {
			resp = fmt.Sprintf("You don't see %v here.\n", args[0].Literal)
		}
	default:
		resp = fmt.Sprintf("You don't see %v here.\n", args[0].Literal)
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
			chLoc.TransferPlayer(ch, loc[dest])
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
		ch.ResponseChannel <- "Get what?\n"
		return nil
	} else {
		itm, err := items.AutoCompleteItems(args[0].Literal, chLoc.Contents)
		if err != nil {
			ch.ResponseChannel <- fmt.Sprintf("You don't see %q here.\n", args[0].Literal)
			return err
		} else {
			chLoc.Remove(itm.ID)
			ch.CharData.Insert(itm)
			chMsg := fmt.Sprintf("You pick up the %s.\n", itm.Name)
			otherMsg := fmt.Sprintf("%s picks up a %s.\n", ch.CharData.Name, itm.Name)
			chLoc.LocalAnnouncePCMsg(ch, chMsg, otherMsg)
			return nil
		}
	}
}

func RunDropCommand(args []Token, ch *chara.ActiveCharacter, loc rooms.RoomList) error {
	chLoc := loc[ch.CharData.Location]
	if len(args) == 0 {
		ch.ResponseChannel <- "Get what?\n"
		return nil
	} else {
		itm, err := items.AutoCompleteItems(args[0].Literal, ch.CharData.Inv)
		if err != nil {
			ch.ResponseChannel <- fmt.Sprintf("You don't have a %q.\n", args[0].Literal)
			return err
		} else {
			ch.CharData.Remove(itm.ID)
			chLoc.Insert(itm)
			chMsg := fmt.Sprintf("You drop the %s on the ground.\n", itm.Name)
			otherMsg := fmt.Sprintf("%s drops a %s on the ground.\n", ch.CharData.Name, itm.Name)
			chLoc.LocalAnnouncePCMsg(ch, chMsg, otherMsg)
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

func RunSayCommand(msg string, ch *chara.ActiveCharacter, loc rooms.RoomList) error {
	chLoc := loc[ch.CharData.Location]
	if msg == "" {
		ch.ResponseChannel <- "Say what?\n"
		return nil
	} else {
		chMsg := fmt.Sprintf("You say %q\n", msg)
		otherMsg := fmt.Sprintf("%s says %q\n", ch.CharData.Name, msg)
		chLoc.LocalAnnouncePCMsg(ch, chMsg, otherMsg)
		return nil
	}
}

func RunTellCommand(args string, ch *chara.ActiveCharacter, charas chara.UserList) error {
	recipName, msg, _ := strings.Cut(args, " ")
	if recipName == "" {
		ch.ResponseChannel <- "Tell who?\n"
		return nil
	} else if msg == "" {
		ch.ResponseChannel <- "Tell them what?\n"
		return nil
	} else {
		charaSlice := []*chara.ActiveCharacter{}
		for _, v := range charas {
			charaSlice = append(charaSlice, v)
		}
		recip, err := chara.AutoCompletePCs(recipName, charaSlice)
		if err != nil {
			ch.ResponseChannel <- "Could not find a player by that name.\n"
			return nil
		}
		chMsg := fmt.Sprintf("You tell %s %q\n", recip.CharData.Name, msg)
		otherMsg := fmt.Sprintf("%s tells you %q\n", ch.CharData.Name, msg)
		ch.ResponseChannel <- chMsg
		recip.ResponseChannel <- otherMsg
		return nil
	}
}
