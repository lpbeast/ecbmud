package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/combat"
	"github.com/lpbeast/ecbmud/commands"
	"github.com/lpbeast/ecbmud/mobs"
	"github.com/lpbeast/ecbmud/rooms"
)

func doServerTick() {
	start := time.Now()
	healTick := tickCounter%200 == 0
	// process everything
	// do mobs - this is just a very basic implementation for now, to get a framework
	// working at all before I try to get more detailed and fancy
	for _, z := range rooms.GlobalZoneList {
		// for each mob, heal if it's a heal tick, then process any actions it might take.
		// for now, that's just whether or not to wander around and autoattacks
		for _, v := range z.ActiveMobs {
			if healTick {
				if v.HPCurrent < v.HPMax {
					v.HPCurrent += 5
					if v.HPCurrent > v.HPMax {
						v.HPCurrent = v.HPMax
					}
				}
				if v.MPCurrent < v.MPMax {
					v.MPCurrent += 5
					if v.MPCurrent > v.MPMax {
						v.MPCurrent = v.MPMax
					}
				}
			}

			mLoc := z.Rooms[v.Loc]
			if len(v.TempInfo.Targets) > 0 {
				if v.TempInfo.AutoAtkCD <= 0 {
					DoCombat(v, v.TempInfo.Targets[0], false)
					if v.TempInfo.Targets[0].GetHP() <= 0 {
						c, ok := v.TempInfo.Targets[0].(*chara.ActiveCharacter)
						if ok {
							chMsg := fmt.Sprintf("\n%s strikes you down!\n", v.GetName())
							otherMsg := fmt.Sprintf("\n%s strikes %s down!\n", v.GetName(), c.GetName())
							mLoc.LocalAnnouncePCMsg(c, chMsg, otherMsg)
							MakePCDead(c)
							if len(v.TempInfo.Targets) == 0 {
								v.ExitCombat()
							}
						}
					}
				} else {
					v.TempInfo.AutoAtkCD -= 1
				}
			}

			if dest, ok := mobWanderDecision(v, mLoc.Exits); ok {
				mLoc.TransferMob(v, dest, true)
			}
		}

		z.RepopCtr += 1
		if z.RepopCtr >= z.RepopTime {
			// make repop times vary slightly by changing where the counter starts.
			// TODO: make sure to bump this up when repop times get increased for actual play.
			z.RepopCtr = rand.Intn(1200) - 599
			z.DoRepop()
		}
	}

	// heal player characters on a 20 second tick
	// TODO: base this on vitality once stats are in
	if healTick {
		for _, v := range chara.GlobalUserList {
			if v.CharData.HPCurrent < v.CharData.HPMax {
				v.CharData.HPCurrent += 5
				if v.CharData.HPCurrent > v.CharData.HPMax {
					v.CharData.HPCurrent = v.CharData.HPMax
				}
			}
			if v.CharData.MPCurrent < v.CharData.MPMax {
				v.CharData.MPCurrent += 5
				if v.CharData.MPCurrent > v.CharData.MPMax {
					v.CharData.MPCurrent = v.CharData.MPMax
				}
			}
		}
	}

	// move on to player commands
	// go through each connected PC one at a time, if they have any commands waiting
	// in the queue, process the first one.
	// this should avoid race conditions even if two characters try to affect the same
	// item or mob on the same tick
	// that is, it'll still be random which player gets to eg take an item, if they try
	// on the same tick, but the code will not end up in a confused or incomplete state
	for _, v := range chara.GlobalUserList {
		if len(v.IncomingCmds) > 0 {
			// response := fmt.Sprintf("DEBUG Server: Received %q from %q\n", v.IncomingCmds[0], v.CharData.Name)
			// fmt.Print(response)
			// v.ResponseChannel <- response
			pc, err := commands.ParseCommand(v.IncomingCmds[0])
			if err != nil {
				log.Println(err.Error())
			}
			if pc != nil {
				err = commands.RunCommand(pc, v)
				if err != nil {
					log.Println(err.Error())
				}
			}
			v.IncomingCmds = v.IncomingCmds[1:]
		}

		// if any characters are in combat, process their autoattacks.
		if len(v.TempInfo.Targets) > 0 {
			if v.TempInfo.AutoAtkCD <= 0 {
				DoCombat(v, v.TempInfo.Targets[0], true)
				if v.TempInfo.Targets[0].GetHP() <= 0 {
					m, ok := v.TempInfo.Targets[0].(*mobs.Mob)
					if ok {
						chLoc := rooms.GlobalZoneList[v.CharData.Zone].Rooms[v.CharData.Location]
						chMsg := fmt.Sprintf("\nYou strike %s down!\n", m.GetName())
						otherMsg := fmt.Sprintf("\n%s strikes %s down!\n", v.GetName(), m.GetName())
						chLoc.LocalAnnouncePCMsg(v, chMsg, otherMsg)
						MakeMobDead(m)
						if len(v.TempInfo.Targets) == 0 {
							v.ExitCombat()
						}
					}
				}
			} else {
				v.TempInfo.AutoAtkCD -= 1
			}
		}

	}

	processingTime := time.Since(start)
	sleepTime := (100 * time.Millisecond) - processingTime
	if sleepTime < 0 {
		log.Printf("Tick length exceeded: %v.\n", sleepTime)
	}
	time.Sleep(sleepTime)
}

// Mob decision-making functions need to have acess to information about the mobs
// and also information about the room they're in. Since rooms need to know about
// the mobs that are in them, this means the mob package can't import the rooms
// package, so decision-making has to get bumped up to a package that can import both.
// This could have gone in the rooms package or a separate mob-control package
// but I think it fits reasonably well here.

func mobWanderDecision(m *mobs.Mob, exits map[string]rooms.TransDest) (string, bool) {
	// 1/300 chance of moving means any given mob should, on average, move once every 30 seconds
	// mobs should not try to wander if they are in combat
	if rand.Intn(300) == 0 && len(m.TempInfo.Targets) == 0 {
		exitSlice := []rooms.TransDest{}
		for _, v := range exits {
			exitSlice = append(exitSlice, v)
		}
		dest := rand.Intn(len(exitSlice))
		// prevent mobs from trying to move out of the zone they're assigned to
		if m.Zone == exitSlice[dest].Zone {
			return exitSlice[dest].Room, true
		}
	}
	return "", false
}

func DoCombat(attacker, defender combat.Combatant, attackerIsPlayer bool) {
	var p *chara.ActiveCharacter
	var ok bool
	if attackerIsPlayer {
		p, ok = attacker.(*chara.ActiveCharacter)
		if !ok {
			fmt.Printf("LOG ERROR: type mismatch in DoCombat\n")
		}
	} else {
		p, ok = defender.(*chara.ActiveCharacter)
		if !ok {
			fmt.Printf("LOG ERROR: type mismatch in DoCombat\n")
		}
	}
	chMsg, otherMsg := attacker.DoAutoAttack()
	chLoc := rooms.GlobalZoneList[p.CharData.Zone].Rooms[p.CharData.Location]
	chLoc.LocalAnnouncePCMsg(p, chMsg, otherMsg)
}

// These functions have to go here to avoid import loops
func MakeMobDead(m *mobs.Mob) {
	mLoc := rooms.GlobalZoneList[m.Zone].Rooms[m.Loc]
	for _, p := range mLoc.PCs {
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
	mZone := rooms.GlobalZoneList[m.Zone]
	for k, v := range mLoc.Mobs {
		if v == m {
			if k == len(mLoc.Mobs)-1 {
				mLoc.Mobs = mLoc.Mobs[:k]
			} else {
				mLoc.Mobs = append(mLoc.Mobs[:k], mLoc.Mobs[k+1:]...)
			}
		}
	}
	mLoc.LocalAnnounce(fmt.Sprintf("\n%s falls over dead!\n", m.GetName()))
	delete(mZone.ActiveMobs, m.ID)
	mZone.DeadMobs[m.ID] = m
}

func MakePCDead(c *chara.ActiveCharacter) {
	chLoc := rooms.GlobalZoneList[c.CharData.Zone].Rooms[c.CharData.Location]
	for _, m := range chLoc.Mobs {
		for i, t := range m.TempInfo.Targets {
			if t == c {
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
	c.ExitCombat()
	chMsg := "\nYou were slain!\nYour consciousness fades, but you wake in a new place...\n"
	otherMsg := fmt.Sprintf("\n%s was slain! They fall to the ground and disappear.\n", c.GetName())
	chLoc.LocalAnnouncePCMsg(c, chMsg, otherMsg)
	c.CharData.HPCurrent = c.CharData.HPMax / 2
	chLoc.TransferPlayer(c, "z1000", "r1000", false)
	commands.RunLookCommand([]commands.Token{}, c)
}
