package mobs

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/lpbeast/ecbmud/items"
)

type Mob struct {
	ID       string   `json:"ID"`
	Name     string   `json:"Name"`
	Keywords []string `json:"Keywords"`
	Desc     string   `json:"Desc"`
	StartLoc string   `json:"StartLoc"`
	ContList []string `json:"ContList"`
	Contents []items.Item
	UUID     string
	Zone     string
	Loc      string
}

type MobList map[string]*Mob

func LoadMobs(zoneID string) (MobList, error) {
	fmt.Printf("Loading mobs for zone %s.\n", zoneID)
	ml := MobList{}
	fname := "mobs" + string(os.PathSeparator) + "mobs-" + zoneID + ".json"
	f, err := os.ReadFile(fname)
	if err != nil {
		fmt.Printf("unable to open items file: %s", err)
		return nil, err
	}

	err = json.Unmarshal(f, &ml)
	if err != nil {
		fmt.Printf("error unmarshaling JSON: %s", err)
		return nil, err
	}
	for _, v := range ml {
		v.UUID = uuid.New().String()
		v.Zone = zoneID
		v.Loc = v.StartLoc
		fmt.Printf("loaded mob: %s: %s\n", v.Name, v.UUID)
	}
	return ml, nil
}

func AutoCompleteMobs(stub string, mobs []*Mob) (*Mob, error) {
	for _, v := range mobs {
		for _, w := range v.Keywords {
			if strings.HasPrefix(w, stub) {
				return v, nil
			}
		}
	}
	return nil, fmt.Errorf("not found: %q", stub)
}
