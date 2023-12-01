package rooms

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/items"
	"github.com/lpbeast/ecbmud/mobs"
)

type TransDest struct {
	Zone        string `json:"Zone"`
	Room        string `json:"Room"`
	IsLocked    bool   `json:"IsLocked"`
	LockKey     string `json:"LockKey"`
	NeedsFlying bool   `json:"NeedsFlying"`
}

type Room struct {
	ID       string `json:"ID"`
	Zone     string
	Name     string               `json:"Name"`
	Desc     string               `json:"Desc"`
	Exits    map[string]TransDest `json:"Exits"`
	ContList []string             `json:"ContList"`
	MobList  []string             `json:"MobList"`
	Contents []items.Item
	Mobs     []*mobs.Mob
	PCs      []*chara.ActiveCharacter
}

type RoomList map[string]*Room

func LoadRooms(zone string, ml mobs.MobList) (RoomList, error) {
	rl := RoomList{}
	fname := "rooms" + string(os.PathSeparator) + zone + ".json"
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
		// set the room's zone to the zone it's being loaded into to avoid mismatches
		// from incorrect data entry
		v.Zone = zone
		// temporary for testing, rooms will be unlikely to just have items lying around
		for _, inum := range v.ContList {
			v.Contents = append(v.Contents, items.GlobalItemList[inum])
		}
	}

	// go through the list of mobs and load them into the correct rooms
	for _, v := range ml {
		rl[v.Loc].Mobs = append(rl[v.Loc].Mobs, v)
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
		if v.ID == itm {
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
		v.SendPrompt()
	}
}

func (r *Room) LocalAnnouncePCMsg(ch *chara.ActiveCharacter, chMsg string, otherMsg string) {
	for _, v := range r.PCs {
		if v == ch {
			v.ResponseChannel <- chMsg
		} else {
			v.ResponseChannel <- otherMsg
		}
		v.SendPrompt()
	}
}

func (r *Room) TransferPlayer(ch *chara.ActiveCharacter, destZone string, destRoom string, announce bool) {
	// Players can't cuurently leave a room while they're fighting, but in future they will
	// be able to flee, recall out, possibly be summoned, and also there may be bugs, so this is
	// for that: remove them from combat and from the aggro tables of all mobs in the room
	for _, m := range r.Mobs {
		for i, t := range m.TempInfo.Targets {
			if t == ch {
				if i == len(m.TempInfo.Targets)-1 {
					m.TempInfo.Targets = m.TempInfo.Targets[:i]
				} else {
					m.TempInfo.Targets = append(m.TempInfo.Targets[:i], m.TempInfo.Targets[i+1:]...)
				}
				if len(m.TempInfo.Targets) == 0 {
					m.ExitCombat()
				}
			}
		}
	}
	ch.ExitCombat()
	// confirm to player that they're going, announce to old room that they're leaving
	// announce to new room that they're arriving before adding them to the room, as the
	// player gets a look around the new room and doesn't need to be told where they came from
	if announce {
		destAnnStr := "somewhere mysterious"
		destAnnStrPC := "somewhere mysterious"
		for k, v := range r.Exits {
			if v.Room == destRoom {
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
		chLeaveMsg := fmt.Sprintf("\nYou travel %s.\n", destAnnStrPC)
		otherLeaveMsg := fmt.Sprintf("\n%s leaves for %s.\n", ch.CharData.Name, destAnnStr)
		r.LocalAnnouncePCMsg(ch, chLeaveMsg, otherLeaveMsg)
		arrAnnStr := "somewhere mysterious"
		for k, v := range GlobalZoneList[destZone].Rooms[destRoom].Exits {
			if v.Room == r.ID {
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
		GlobalZoneList[destZone].Rooms[destRoom].LocalAnnounce(fmt.Sprintf("\n%s arrives from %s.\n", ch.CharData.Name, arrAnnStr))
	}

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
	GlobalZoneList[destZone].Rooms[destRoom].PCs = append(GlobalZoneList[destZone].Rooms[destRoom].PCs, ch)
	ch.CharData.Zone = destZone
	ch.CharData.Location = destRoom
}

// Mobs do not wander into other zones.
func (r *Room) TransferMob(m *mobs.Mob, destRoom string, announce bool) {
	// Mobs can't wander while they're in combat, so for now this is just a backstop.
	// At some future point mobs may be able to escape combat through some means,
	// or be summoned, or something, so just to be sure, when a mob moves to a different room,
	// we wipe its aggro table and remove it from player target lists.
	for _, p := range r.PCs {
		for i, t := range p.TempInfo.Targets {
			if t == m {
				if i == len(p.TempInfo.Targets)-1 {
					p.TempInfo.Targets = p.TempInfo.Targets[:i]
				} else {
					p.TempInfo.Targets = append(p.TempInfo.Targets[:i], p.TempInfo.Targets[i+1:]...)
				}
				if len(p.TempInfo.Targets) == 0 {
					p.ExitCombat()
				}
			}
		}
	}
	m.ExitCombat()
	if announce {
		destAnnStr := "somewhere mysterious"
		for k, v := range r.Exits {
			if v.Room == destRoom {
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
		r.LocalAnnounce(fmt.Sprintf("\n%s leaves for %s.\n", m.Name, destAnnStr))

		arrAnnStr := "somewhere mysterious"
		for k, v := range GlobalZoneList[m.Zone].Rooms[destRoom].Exits {
			if v.Room == r.ID {
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
		GlobalZoneList[m.Zone].Rooms[destRoom].LocalAnnounce(fmt.Sprintf("\n%s arrives from %s.\n", m.Name, arrAnnStr))
	}

	// remove mob from old room, add them to new room
	for k, v := range r.Mobs {
		if v == m {
			if k == len(r.Mobs)-1 {
				r.Mobs = r.Mobs[:k]
			} else {
				r.Mobs = append(r.Mobs[:k], r.Mobs[k+1:]...)
			}
		}
	}
	GlobalZoneList[m.Zone].Rooms[destRoom].Mobs = append(GlobalZoneList[m.Zone].Rooms[destRoom].Mobs, m)
	// update mob's location as well as room moblists or it tries to leave from the same
	// room over and over and multiplies
	m.Loc = destRoom
}
