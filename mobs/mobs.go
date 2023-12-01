package mobs

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/lpbeast/ecbmud/combat"
	"github.com/lpbeast/ecbmud/items"
)

type Transients struct {
	Targets   []combat.Combatant
	AutoAtkCD int
}

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

	HPCurrent int `json:"HPCurrent"`
	HPMax     int `json:"HPMax"`
	MPCurrent int `json:"MPCurrent"`
	MPMax     int `json:"MPMax"`
	AtkRoll   int `json:"AtkRoll"`
	DamRoll   int `json:"DamRoll"`

	TempInfo Transients
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

func (m *Mob) EnterCombat(target combat.Combatant) {
	m.TempInfo.AutoAtkCD = 1
	m.TempInfo.Targets = append(m.TempInfo.Targets, target)
}

func (m *Mob) DoAutoAttack() (string, string) {
	m.TempInfo.AutoAtkCD = 20
	chAtkMsg := fmt.Sprintf("\n%s swings at you.\n", m.GetName())
	otherAtkMsg := fmt.Sprintf("\n%s swings at %s.\n", m.GetName(), m.TempInfo.Targets[0].GetName())
	tn := 99 - m.TempInfo.Targets[0].GetDefense()
	if rand.Intn(100)+m.AtkRoll <= tn {
		dmg := rand.Intn(10) + 1 + m.DamRoll
		m.TempInfo.Targets[0].ReceiveDamage(dmg)
		chAtkMsg += fmt.Sprintf("%s hits you for %d damage!\n", m.GetName(), dmg)
		otherAtkMsg += fmt.Sprintf("%s hits %s for %d damage.\n", m.GetName(), m.TempInfo.Targets[0].GetName(), dmg)
	} else {
		chAtkMsg += fmt.Sprintf("%s misses you.\n", m.GetName())
		otherAtkMsg += fmt.Sprintf("%s misses %s.\n", m.GetName(), m.TempInfo.Targets[0].GetName())
	}
	return chAtkMsg, otherAtkMsg
}

func (m *Mob) ReceiveDamage(dmg int) {
	m.HPCurrent -= dmg
}

func (m *Mob) GetName() string {
	return m.Name
}

func (m *Mob) GetDefense() int {
	return 0
}

func (m *Mob) GetHP() int {
	return m.HPCurrent
}

func (m *Mob) ExitCombat() {
	m.TempInfo.Targets = []combat.Combatant{}
}
