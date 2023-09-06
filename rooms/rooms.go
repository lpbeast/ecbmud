package rooms

import (
	"encoding/json"
	"fmt"
	"os"
)

type Room struct {
	ID    string            `json:"ID"`
	Name  string            `json:"Name"`
	Desc  string            `json:"Desc"`
	Exits map[string]string `json:"Exits"`
}

type RoomList map[string]*Room

func LoadRooms() (RoomList, error) {
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
	return rl, nil
}
