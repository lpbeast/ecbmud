package rooms

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/items"
)

type Room struct {
	ID       string            `json:"ID"`
	Name     string            `json:"Name"`
	Desc     string            `json:"Desc"`
	Exits    map[string]string `json:"Exits"`
	ContList []string          `json:"ContList"`
	Contents []items.Item
	PCs      []*chara.ActiveCharacter
}

type RoomList map[string]*Room

func LoadRooms() (RoomList, error) {
	il, err := items.LoadItems()
	if err != nil {
		fmt.Printf("unable to load items")
		return nil, err
	}

	rl := RoomList{}
	fname := "rooms/rooms.json"
	f, err := os.ReadFile(fname)
	if err != nil {
		fmt.Printf("unable to open rooms file: %s", err)
		return nil, err
	}

	err = json.Unmarshal(f, &rl)
	if err != nil {
		fmt.Printf("error unmarshaling JSON: %s", err)
		return nil, err
	}

	for _, v := range rl {
		for _, inum := range v.ContList {
			v.Contents = append(v.Contents, il[inum])
		}
		fmt.Printf("room %s: contents %+v\n", v.ID, v.Contents)
	}
	return rl, nil
}

func (r *Room) ListContents() []string {
	itemList := []string{}
	for _, v := range r.Contents {
		itemList = append(itemList, v.Name)
	}
	return itemList
}

func (r *Room) Insert(itm items.Item) {
	r.Contents = append(r.Contents, itm)
}

func (r *Room) Remove(itm string) error {
	for k, v := range r.Contents {
		if v.Serial == itm {
			if k == len(r.Contents)-1 {
				r.Contents = r.Contents[:k]
			} else {
				r.Contents = append(r.Contents[:k], r.Contents[k+1:]...)
			}
			return nil
		}
	}
	return fmt.Errorf("not found: %q", itm)
}

func (r *Room) LocalAnnounce(msg string) {
	for _, v := range r.PCs {
		v.ResponseChannel <- msg
	}
}

func (r *Room) LocalAnnouncePCMsg(ch *chara.ActiveCharacter, chMsg string, otherMsg string) {
	for _, v := range r.PCs {
		if v == ch {
			v.ResponseChannel <- chMsg
		} else {
			v.ResponseChannel <- otherMsg
		}
	}
}

func (r *Room) TransferPlayer(ch *chara.ActiveCharacter, dest *Room) {
	// confirm to player that they're going, announce to old room that they're leaving
	// announce to new room that they're arriving before adding them to the room, as the
	// player gets a look around the new room and doesn't need to be told where they came from
	destAnnStr := "somewhere mysterious"
	destAnnStrPC := "somewhere mysterious"
	for k, v := range r.Exits {
		if v == dest.ID {
			destAnnStrPC = k
			switch k {
			case "up":
				destAnnStr = "above"
			case "down":
				destAnnStr = "below"
			default:
				destAnnStr = "the " + k
			}
		}
	}
	chLeaveMsg := fmt.Sprintf("You travel %s", destAnnStrPC)
	otherLeaveMsg := fmt.Sprintf("%s leaves for %s.\n", ch.CharData.Name, destAnnStr)
	r.LocalAnnouncePCMsg(ch, chLeaveMsg, otherLeaveMsg)
	arrAnnStr := "somewhere mysterious"
	for k, v := range dest.Exits {
		if v == r.ID {
			switch k {
			case "up":
				arrAnnStr = "above"
			case "down":
				arrAnnStr = "below"
			default:
				arrAnnStr = "the " + k
			}
		}
	}
	dest.LocalAnnounce(fmt.Sprintf("%s arrives from %s.\n", ch.CharData.Name, arrAnnStr))

	// remove character from old room, add them to new room
	for k, v := range r.PCs {
		if v == ch {
			if k == len(r.PCs)-1 {
				r.PCs = r.PCs[:k]
			} else {
				r.PCs = append(r.PCs[:k], r.PCs[k+1:]...)
			}
		}
	}
	dest.PCs = append(dest.PCs, ch)
}
