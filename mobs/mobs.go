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
	Loc      string   `json:"Loc"`
	ContList []string `json:"ContList"`
	Contents []items.Item
	UUID     string
}

type MobList map[string]*Mob

func LoadMobs() (MobList, error) {
	ml := MobList{}
	fname := "mobs/mobs.json"
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
		fmt.Printf("loaded mob: %s: %s\n", v.Name, v.UUID)
	}
	return ml, nil
}

func AutoCompleteMobs(stub string, mobs []*Mob) (Mob, error) {
	for _, v := range mobs {
		for _, w := range v.Keywords {
			if strings.HasPrefix(w, stub) {
				return *v, nil
			}
		}
	}
	return Mob{}, fmt.Errorf("not found: %q", stub)
}
