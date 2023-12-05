package main

import (
	"bufio"
	"crypto/sha512"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/combat"
	"github.com/lpbeast/ecbmud/commands"
	"github.com/lpbeast/ecbmud/items"
	"github.com/lpbeast/ecbmud/rooms"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type inputMsg struct {
	chara string
	input string
}

type ctrlMsg struct {
	chara         string
	event         string
	returnChannel chan string
}

// createConnection sets up a handler for I/O for a given connection and character
func createConnection(c net.Conn, servChan chan inputMsg, ctrlChan chan ctrlMsg) {
	ch := make(chan string)
	ic := make(chan string, 20)
	loginChan := make(chan string)
	sc := bufio.NewScanner(c)
	loggedIn := false
	name := ""

	io.WriteString(c, "Welcome to Endless Crystal Blue MUD\n")

	go chara.DoLogin(ch, loginChan)

	go func(c net.Conn, inputChan chan string) {
		for sc.Scan() {
			line := sc.Text()
			// fmt.Printf("DEBUG Handling input line %q.\n", line)
			for len(inputChan) >= cap(inputChan) {
				time.Sleep(100 * time.Millisecond)
			}
			inputChan <- line
		}
	}(c, ic)

	for !loggedIn {
		select {
		case response := <-ch:
			// fmt.Printf("DEBUG Handler received control message: %q\n", response)
			if response[:8] == "Success:" {
				name = response[8:]
				loggedIn = true
			} else {
				io.WriteString(c, response)
			}
		case input := <-ic:
			loginChan <- input
		default:
		}
	}
	// fmt.Printf("DEBUG Handler for %q sending LOGIN\n", name)
	eventMsg := ctrlMsg{name, "LOGIN", ch}
	ctrlChan <- eventMsg

	connected := true
	for connected {
		select {
		case input := <-ic:
			msgForServer := inputMsg{name, input}
			servChan <- msgForServer
		case resp, ok := <-ch:
			if !ok {
				connected = false
			} else {
				io.WriteString(c, resp)
			}
		default:
		}
	}

	c.Close()
}

func serverCleanup() {
	fmt.Printf("Shutting down server.\n")
}

func checkCharFile(lfname string) error {
	// if the character list does not exist, create it, and go through generating an
	// admin character. Admin character has no special powers yet but will in future.
	// If the file exists but is empty, no character will be created, and any desired
	// admin characters will need to be created by the normal process and promoted
	// manually.
	_, err := os.Stat(lfname)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Welcome to Endless Crystal Blue MUD setup.\n")
		fmt.Printf("Enter a name for your admin character.\n")
		var name, pw1, pw2 string
		fmt.Scanln(&name)
		name = cases.Title(language.English).String(name)
		for pw1 != pw2 || pw1 == "" || pw2 == "" {
			fmt.Printf("Enter a password for this character.\n")
			fmt.Scanln(&pw1)
			fmt.Printf("Enter it again to confirm.\n")
			fmt.Scanln(&pw2)
		}
		fmt.Print("Creating character file with initial character.\n")

		lf, err := os.OpenFile(lfname, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer lf.Close()

		hasher := sha512.New()
		pwHash := fmt.Sprintf("%x", hasher.Sum([]byte(pw1)))
		newCharEntry := []string{name, pwHash}

		if err != nil {
			return err
		}
		w := csv.NewWriter(lf)
		if err := w.Write(newCharEntry); err != nil {
			return err
		}
		w.Flush()

		charFileName := "chara" + string(os.PathSeparator) + name + ".json"
		cf, err := os.OpenFile(charFileName, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatal(err)
		}
		newCharSheet := chara.CharSheet{
			Name:      name,
			Zone:      "z1000",
			Location:  "r1000",
			Desc:      "A formless being.\n",
			HPCurrent: 100,
			HPMax:     100,
			MPCurrent: 100,
			MPMax:     100,
			AtkRoll:   0,
			DamRoll:   0,
			Inv:       []items.Item{},
		}
		jChar, err := json.MarshalIndent(newCharSheet, "", "\t")
		if err != nil {
			log.Fatal(err)
		}
		cf.Write(jChar)
		return nil
	}
	return err
}

var tickCounter = 0

func main() {
	defer serverCleanup()

	err := checkCharFile(chara.CharListFile)
	if err != nil {
		log.Fatalf("Unable to find or create characters file: %s\n", err.Error())
	}

	runWorld := true
	// servChan is for the connection handlers to send user input to the main server
	servChan := make(chan inputMsg, 400)
	// connChan is for the goroutine that listens for new connections to tell the server
	// that there's a new connection, and to hand the connection over to the connection handler
	connChan := make(chan net.Conn, 20)
	// ctrlChan is for the connection handlers to send control messages like LOGIN and QUIT
	// to the main server
	ctrlChan := make(chan ctrlMsg, 20)
	// GlobalUserList associates character names with information about that character
	chara.GlobalUserList = chara.UserList{}

	l, err := net.Listen("tcp", ":4040")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	go func(connChan chan net.Conn) {
		fmt.Printf("Connection dispatcher started.\n")
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}
			connChan <- conn
		}
	}(connChan)

	fmt.Printf("Loading item templates.\n")
	err = items.LoadItems()
	if err != nil {
		log.Fatal(err)
	}
	if items.GlobalItemList == nil {
		log.Fatal("No rooms loaded.\n")
	}

	fmt.Printf("Loading zones.\n")
	err = rooms.LoadZones()
	if err != nil {
		log.Fatal(err)
	}
	if rooms.GlobalZoneList == nil {
		log.Fatal("No rooms loaded.\n")
	}

	fmt.Printf("Server starting.\n")
	for runWorld {
		tickCounter++
		// process all connection attempts and user input that has come in since the last tick
		for len(connChan) > 0 || len(ctrlChan) > 0 || len(servChan) > 0 {
			select {
			case conn := <-connChan:
				// fmt.Printf("DEBUG Received connection.\n")
				go createConnection(conn, servChan, ctrlChan)
			case incoming := <-ctrlChan:
				// fmt.Printf("DEBUG Received control message.\n")
				switch incoming.event {
				case "LOGIN":
					fmt.Printf("LOG %v Server received LOGIN for %q\n", time.Now(), incoming.chara)
					// TODO: pull this into a function
					if chara.GlobalUserList[incoming.chara] == nil {
						charFile := "chara" + string(os.PathSeparator) + incoming.chara + ".json"
						charSheet := chara.CharSheet{}
						cf, err := os.ReadFile(charFile)
						if err != nil {
							incoming.returnChannel <- fmt.Sprintf("Could not read character file for %q.\n", incoming.chara)
							close(incoming.returnChannel)
						}
						err = json.Unmarshal(cf, &charSheet)
						if err != nil {
							incoming.returnChannel <- fmt.Sprintf("error unmarshaling JSON: %s", err)
							close(incoming.returnChannel)
						}

						transients := chara.Transients{Position: chara.STANDING, Targets: []combat.Combatant{}}

						charToLogIn := chara.ActiveCharacter{ResponseChannel: incoming.returnChannel, Cooldown: 0, CharData: charSheet, TempInfo: transients, IncomingCmds: []string{}}
						chara.GlobalUserList[incoming.chara] = &charToLogIn
						incoming.returnChannel <- fmt.Sprintf("Welcome to Endless Crystal Blue MUD, %s.\n", incoming.chara)
						pcZone := charToLogIn.CharData.Zone
						pcRoom := charToLogIn.CharData.Location
						rooms.GlobalZoneList[pcZone].Rooms[pcRoom].PCs = append(rooms.GlobalZoneList[pcZone].Rooms[pcRoom].PCs, &charToLogIn)
						rooms.GlobalZoneList[pcZone].Rooms[pcRoom].LocalAnnounce(fmt.Sprintf("%s wakes up.\n", charToLogIn.CharData.Name))
						commands.RunLookCommand([]commands.Token{}, &charToLogIn)
					} else {
						incoming.returnChannel <- "Character already logged in.\n"
						chara.GlobalUserList[incoming.chara].ResponseChannel <- "Duplicate login attempt.\n"
						close(incoming.returnChannel)
					}
				default:
					log.Fatalf("Unexpected control message %q\n", incoming.event)
				}
			case incoming := <-servChan:
				// fmt.Printf("DEBUG Received input message on tick %v.\n", tickCounter)
				chara.GlobalUserList[incoming.chara].IncomingCmds = append(chara.GlobalUserList[incoming.chara].IncomingCmds, incoming.input)
			default:
			}
		}
		// TODO: at some point this will have to be modified so that it distinguishes
		// between "the list of possible mobs" and "the list of currently spawned and
		// active mobs" but that can wait till I get mobs working at all and know what
		// I have to work with there.
		doServerTick()
	}
}
