package rooms

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lpbeast/ecbmud/items"
)

type Room struct {
	ID       string            `json:"ID"`
	Name     string            `json:"Name"`
	Desc     string            `json:"Desc"`
	Exits    map[string]string `json:"Exits"`
	ContList []string          `json:"ContList"`
	Contents []items.Item
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
