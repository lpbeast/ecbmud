package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/commands"
	"github.com/lpbeast/ecbmud/mobs"
	"github.com/lpbeast/ecbmud/rooms"
)

func doServerTick(world rooms.RoomList, users chara.UserList, mobs mobs.MobList) {
	start := time.Now()
	// process everything
	// do mobs - this is just a very basic implementation for now, to get a framework
	// working at all before I try to get more detailed and fancy
	for _, v := range mobs {
		mLoc := world[v.Loc]
		if dest, ok := mobWanderDecision(v, mLoc.Exits); ok {
			mLoc.TransferMob(v, world[dest])
		}
	}

	// move on to player commands
	// go through each connected PC one at a time, if they have any commands waiting
	// in the queue, process the first one.
	// this should avoid race conditions even if two characters try to affect the same
	// item or mob on the same tick
	for _, v := range users {
		if len(v.IncomingCmds) > 0 {
			response := fmt.Sprintf("Server: Received %q from %q\n", v.IncomingCmds[0], v.CharData.Name)
			fmt.Print(response)
			v.ResponseChannel <- response
			pc, err := commands.ParseCommand(v.IncomingCmds[0])
			if err != nil {
				log.Println(err.Error())
			}
			err = commands.RunCommand(pc, v, world, users)
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
	// if rand.Intn(100) == 0 {
	// 	for _, v := range users {
	// 		v.ResponseChannel <- "Random asynchronous event!\n"
	// 		break
	// 	}
	// }
}

// Mob decision-making functions need to have acess to information about the mobs
// and also information about the room they're in. Since rooms need to know about
// the mobs that are in them, this means the mob package can't import the rooms
// package, so decision-making has to get bumped up to a package that can import both.
// This could have gone in the rooms package or a separate mob-control package
// but I think it fits reasonably well here.

func mobWanderDecision(m *mobs.Mob, exits map[string]string) (string, bool) {
	// 1/300 chance of moving means any given mob should, on average, move once every 30 seconds
	if rand.Intn(300) == 0 {
		exitSlice := []string{}
		for _, v := range exits {
			exitSlice = append(exitSlice, v)
		}
		dest := rand.Intn(len(exitSlice))
		return exitSlice[dest], true
	}
	return "", false
}
