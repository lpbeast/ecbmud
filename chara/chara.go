package chara

import (
	"crypto/sha512"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"unicode"

	"github.com/lpbeast/ecbmud/combat"
	"github.com/lpbeast/ecbmud/items"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const CharListFile = "chara/charlist.csv"

const (
	STANDING = iota
	FIGHTING
	SITTING
	SLEEPING
)

var invalidNames = map[string]string{
	"new":  "new",
	"quit": "quit",
	"look": "look",
	"save": "save",
	"get":  "get",
	"kill": "kill",
	"cast": "cast",
}

type CharSheet struct {
	Name     string `json:"Name"`
	Zone     string `json:"Zone"`
	Location string `json:"Location"`
	Desc     string `json:"Desc"`

	HPCurrent int `json:"HPCurrent"`
	HPMax     int `json:"HPMax"`
	MPCurrent int `json:"MPCurrent"`
	MPMax     int `json:"MPMax"`
	AtkRoll   int `json:"AtkRoll"`
	DamRoll   int `json:"DamRoll"`

	Inv []items.Item
}

type Transients struct {
	Position  int
	Targets   []combat.Combatant
	AutoAtkCD int
}

type ActiveCharacter struct {
	ResponseChannel chan string
	Cooldown        int
	CharData        CharSheet
	TempInfo        Transients
	IncomingCmds    []string
}

type UserList map[string]*ActiveCharacter

var GlobalUserList UserList

func checkValidName(name string, nameList map[string]string, invalidNames map[string]string) bool {
	if len(name) > 16 {
		return false
	}
	if len(name) < 3 {
		return false
	}
	for _, v := range name {
		if !unicode.IsLetter(v) {
			return false
		}
	}
	return nameList[name] == "" && invalidNames[strings.ToLower(name)] == ""
}

func checkValidPW(pw string) bool {
	if len(pw) > 64 {
		return false
	}
	if len(pw) < 8 {
		return false
	}
	return true
}

func getNameList(fname string) (map[string]string, error) {
	nameList := make(map[string]string)
	f, err := os.OpenFile(fname, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	nameSlice, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	for _, v := range nameSlice {
		nameList[v[0]] = v[1]
	}
	return nameList, nil
}

func DoLogin(ch chan string, loginChan chan string) {
	name := ""
	loggedIn := false
	for !loggedIn {
		ch <- "Enter a character name to log in, or 'new' to create a character.\nName: "
		name = <-loginChan
		if strings.ToLower(name) == "new" {
			create(ch, loginChan)
			loggedIn = true
		} else {
			name = cases.Title(language.English).String(name)
			nameList, err := getNameList(CharListFile)
			if err != nil {
				log.Fatal(err)
			}
			storedHash := nameList[name]
			ch <- "Password: "
			sentPW := <-loginChan
			hasher := sha512.New()
			sentHash := fmt.Sprintf("%x", hasher.Sum([]byte(sentPW)))
			if sentHash == storedHash {
				loggedIn = true
			}
		}
	}
	ch <- fmt.Sprintf("Success:%s", name)
}

func create(ch chan string, createChan chan string) {
	var name, pw1, pw2 string
	// get the list of existing names so we can check if a name is available
	// nameList is a map from character name to hashed password
	nameList, err := getNameList(CharListFile)
	if err != nil {
		log.Fatal(err)
	}
	ready := false
	ch <- "Names must be between 3 and 16 letters.\n"
	ch <- "Do not use numbers, punctuation, spaces, MUD commands, or offensive words.\n"
	for !ready {
		ch <- "Enter a name for your character.\n"
		name = <-createChan
		name = cases.Title(language.English).String(name)
		ready = checkValidName(name, nameList, invalidNames)
	}
	pwready := false
	for !pwready {
		pw1ready := false
		ch <- "Passwords must be between 8 and 64 characters long.\n"
		for !pw1ready {
			ch <- "Enter a password for your character.\n"
			pw1 = <-createChan
			pw1ready = checkValidPW(pw1)
		}
		ch <- "Confirm your password.\n"
		pw2 = <-createChan
		pwready = (pw1 == pw2)
	}

	hasher := sha512.New()
	pwHash := fmt.Sprintf("%x", hasher.Sum([]byte(pw1)))
	newCharEntry := []string{name, pwHash}

	listFile, err := os.OpenFile(CharListFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer listFile.Close()
	w := csv.NewWriter(listFile)
	if err := w.Write(newCharEntry); err != nil {
		log.Fatal(err)
	}
	w.Flush()
	charFileName := "chara" + string(os.PathSeparator) + name + ".json"
	cf, err := os.OpenFile(charFileName, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	newCharSheet := CharSheet{name, "z1000", "r1000", "A formless being.\n", 100, 100, 100, 100, 0, 0, []items.Item{}}
	jChar, err := json.MarshalIndent(newCharSheet, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	cf.Write(jChar)
	ch <- fmt.Sprintf("Success:%s", name)
}

func (c *CharSheet) ListContents() []string {
	itemList := []string{}
	for _, v := range c.Inv {
		itemList = append(itemList, v.Name)
	}
	return itemList
}

func (c *CharSheet) Insert(itm items.Item) {
	c.Inv = append(c.Inv, itm)
}

func (c *CharSheet) Remove(itm string) error {
	for k, v := range c.Inv {
		if v.ID == itm {
			if k == len(c.Inv)-1 {
				c.Inv = c.Inv[:k]
			} else {
				c.Inv = append(c.Inv[:k], c.Inv[k+1:]...)
			}
			return nil
		}
	}
	return fmt.Errorf("not found: %q", itm)
}

func AutoCompletePCs(stub string, chList []*ActiveCharacter) (*ActiveCharacter, error) {
	for _, v := range chList {
		if strings.HasPrefix(strings.ToLower(v.CharData.Name), stub) {
			return v, nil
		}
	}
	return nil, fmt.Errorf("not found: %q", stub)
}

func (c *ActiveCharacter) SendPrompt() {
	p := fmt.Sprintf("\n%d/%d HP %d/%d MP >>", c.CharData.HPCurrent, c.CharData.HPMax, c.CharData.MPCurrent, c.CharData.MPMax)
	c.ResponseChannel <- p
}

func (c *ActiveCharacter) EnterCombat(target combat.Combatant) {
	c.TempInfo.AutoAtkCD = 0
	c.TempInfo.Position = FIGHTING
	c.TempInfo.Targets = append(c.TempInfo.Targets, target)
}

// The function for announcing messages to a room is in the rooms package, which
// imports this package, so this has to assemble the message strings and return them,
// rather than making the announcement itself.
func (c *ActiveCharacter) DoAutoAttack() (string, string) {
	c.TempInfo.AutoAtkCD = 20
	chAtkMsg := fmt.Sprintf("\nYou swing at %s.\n", c.TempInfo.Targets[0].GetName())
	otherAtkMsg := fmt.Sprintf("\n%s swings at %s.\n", c.GetName(), c.TempInfo.Targets[0].GetName())
	tn := 99 - c.TempInfo.Targets[0].GetDefense()
	if rand.Intn(100)+c.CharData.AtkRoll <= tn {
		dmg := rand.Intn(10) + 1 + c.CharData.DamRoll
		c.TempInfo.Targets[0].ReceiveDamage(dmg)
		chAtkMsg += fmt.Sprintf("You hit %s for %d damage!\n", c.TempInfo.Targets[0].GetName(), dmg)
		otherAtkMsg += fmt.Sprintf("%s hits %s for %d damage.\n", c.GetName(), c.TempInfo.Targets[0].GetName(), dmg)
	} else {
		chAtkMsg += fmt.Sprintf("You miss %s.\n", c.TempInfo.Targets[0].GetName())
		otherAtkMsg += fmt.Sprintf("%s misses %s.\n", c.GetName(), c.TempInfo.Targets[0].GetName())
	}
	return chAtkMsg, otherAtkMsg
}

func (c *ActiveCharacter) ReceiveDamage(dmg int) {
	c.CharData.HPCurrent -= dmg
}

func (c *ActiveCharacter) GetName() string {
	return c.CharData.Name
}

func (c *ActiveCharacter) GetDefense() int {
	return 0
}

func (c *ActiveCharacter) GetHP() int {
	return c.CharData.HPCurrent
}

func (c *ActiveCharacter) ExitCombat() {
	c.TempInfo.Position = STANDING
	c.TempInfo.Targets = []combat.Combatant{}
}

func (c *ActiveCharacter) Save() error {
	// this is not how it should be, BUT when i try to do it properly by saving a
	// temp file and then renaming it over the original save, it claims that the
	// file is in use by another process. But I can open it and just write over it
	// with no problem?
	// TODO: figure out what's going on with this and fix it, so that saves have
	// less risk of getting corrupted.
	// tempFileName := "chara" + string(os.PathSeparator) + c.GetName() + ".tmp"
	charFileName := "chara" + string(os.PathSeparator) + c.GetName() + ".json"
	cf, err := os.OpenFile(charFileName, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	jChar, err := json.MarshalIndent(c.CharData, "", "\t")
	if err != nil {
		return err
	}
	_, err = cf.Write(jChar)
	if err != nil {
		return err
	}
	// err = os.Rename(tempFileName, charFileName)
	// done now, so return err whether or not it's nil. ideally it should be nil.
	return err
}
