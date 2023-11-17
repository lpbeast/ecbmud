package rooms

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lpbeast/ecbmud/mobs"
)

type ZoneTemplate struct {
	ID       string   `json:"ID"`
	Name     string   `json:"Name"`
	MobsList []string `json:"MobsList"`
}

type ZoneTemplateList map[string]ZoneTemplate

type Zone struct {
	ID         string
	Name       string
	Rooms      RoomList
	ActiveMobs mobs.MobList
	DeadMobs   mobs.MobList
}

type ZoneList map[string]*Zone

// Just like with items, it's easier to just have a global zones list than to
// pass it down through layers of calls.
var GlobalZoneList ZoneList

func LoadZones() error {
	ztl := ZoneTemplateList{}
	GlobalZoneList = ZoneList{}
	fname := "rooms/zones.json"
	f, err := os.ReadFile(fname)
	if err != nil {
		fmt.Printf("unable to open zones file: %s", err)
		return err
	}

	err = json.Unmarshal(f, &ztl)
	if err != nil {
		fmt.Printf("error unmarshaling JSON: %s", err)
		return err
	}

	for _, v := range ztl {
		ml, err := mobs.LoadMobs(v.ID)
		if err != nil {
			fmt.Printf("error loading mobs for zone %s: %s.\n", v.ID, err)
			return err
		}

		rl, err := LoadRooms(v.ID, ml)
		if err != nil {
			fmt.Printf("error loading rooms for zone %s: %s.\n", v.ID, err)
			return err
		}

		z := Zone{
			ID:         v.ID,
			Name:       v.Name,
			Rooms:      rl,
			ActiveMobs: ml,
		}
		GlobalZoneList[v.ID] = &z
	}

	return nil
}
