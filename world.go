package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/commands"
	"github.com/lpbeast/ecbmud/rooms"
)

func doServerTick(world rooms.RoomList, users chara.UserList) {
	start := time.Now()
	// process everything
	// do mobs once they're implemented

	// move on to player commands
	for _, v := range users {
		if len(v.IncomingCmds) > 0 {
			response := fmt.Sprintf("Server: Received %q from %q\n", v.IncomingCmds[0], v.CharData.Name)
			fmt.Print(response)
			v.ResponseChannel <- response
			pc, err := commands.ParseCommand(v.IncomingCmds[0])
			if err != nil {
				log.Println(err.Error())
			}
			err = commands.RunCommand(pc, v, world)
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
	if rand.Intn(100) == 0 {
		for _, v := range users {
			v.ResponseChannel <- "Random asynchronous event!\n"
			break
		}
	}
}
