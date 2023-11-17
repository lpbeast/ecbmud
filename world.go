package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/commands"
	"github.com/lpbeast/ecbmud/mobs"
	"github.com/lpbeast/ecbmud/rooms"
)

func doServerTick() {
	start := time.Now()
	// process everything
	// do mobs - this is just a very basic implementation for now, to get a framework
	// working at all before I try to get more detailed and fancy
	for _, z := range rooms.GlobalZoneList {
		// for each mob, decide whether or not to have it wander around
		for _, v := range z.ActiveMobs {
			mLoc := z.Rooms[v.Loc]
			if dest, ok := mobWanderDecision(v, mLoc.Exits); ok {
				mLoc.TransferMob(v, dest)
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
			err = commands.RunCommand(pc, v)
			if err != nil {
				log.Println(err.Error())
			}
			v.IncomingCmds = v.IncomingCmds[1:]
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
	if rand.Intn(300) == 0 {
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
