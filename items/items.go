package items

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Container interface {
	ListContents() []string
	Insert(itm string)
	Remove(itm string) error
}

type Item struct {
	ID       string   `json:"ID"`
	Name     string   `json:"Name"`
	Keywords []string `json:"Keywords"`
	Desc     string   `json:"Desc"`
}

type ItemList map[string]Item

// There's only one item list needed, and several things need access to it. A global
// variable available to the whole server reduces the number of arguments that have
// to be passed down through layers of function calls.
var GlobalItemList ItemList

func LoadItems() error {
	GlobalItemList = ItemList{}
	fname := "items/items.json"
	f, err := os.ReadFile(fname)
	if err != nil {
		fmt.Printf("unable to open items file: %s", err)
		return err
	}

	err = json.Unmarshal(f, &GlobalItemList)
	if err != nil {
		fmt.Printf("error unmarshaling JSON: %s", err)
		return err
	}
	return nil
}

func AutoCompleteItems(stub string, items []Item) (Item, error) {
	for _, v := range items {
		for _, w := range v.Keywords {
			if strings.HasPrefix(w, stub) {
				return v, nil
			}
		}
	}
	return Item{}, fmt.Errorf("not found: %q", stub)
}
