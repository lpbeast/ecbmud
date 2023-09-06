package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/lpbeast/ecbmud/chara"
	"github.com/lpbeast/ecbmud/rooms"
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
			fmt.Printf("Handling input line %q.\n", line)
			for len(inputChan) >= cap(inputChan) {
				time.Sleep(100 * time.Millisecond)
			}
			inputChan <- line
		}
	}(c, ic)

	for !loggedIn {
		select {
		case response := <-ch:
			fmt.Printf("Handler received control message: %q\n", response)
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
	fmt.Printf("Handler for %q sending LOGIN\n", name)
	eventMsg := ctrlMsg{name, "LOGIN", ch}
	ctrlChan <- eventMsg

	connected := true
	for connected {
		select {
		case input := <-ic:
			if input == "QUIT" {
				eventMsg := ctrlMsg{name, "QUIT", ch}
				ctrlChan <- eventMsg
				connected = false
			} else {
				fmt.Printf("Putting %q on the server channel.\n", input)
				msgForServer := inputMsg{name, input}
				servChan <- msgForServer
			}
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

func main() {
	runWorld := true
	// servChan is for the connection handlers to send user input to the main server
	servChan := make(chan inputMsg, 400)
	// connChan is for the goroutine that listens for new connections to tell the server
	// that there's a new connection, and to hand the connection over to the connection handler
	connChan := make(chan net.Conn, 20)
	// ctrlChan is for the connection handlers to send control messages like LOGIN and QUIT
	// to the main server
	ctrlChan := make(chan ctrlMsg, 20)
	// activeUsers associates character names with information about that character
	activeUsers := chara.UserList{}

	tickCounter := 0

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

	fmt.Printf("Loading rooms.\n")
	worldRooms, err := rooms.LoadRooms()
	if err != nil {
		log.Fatal(err)
	}
	if worldRooms == nil {
		log.Fatal("No rooms loaded.\n")
	}

	fmt.Printf("Server starting.\n")
	for runWorld {
		tickCounter++
		// process all connection attempts and user input that has come in since the last tick
		for len(connChan) > 0 || len(ctrlChan) > 0 || len(servChan) > 0 {
			select {
			case conn := <-connChan:
				fmt.Printf("Received connection.\n")
				go createConnection(conn, servChan, ctrlChan)
			case incoming := <-ctrlChan:
				fmt.Printf("Received control message.\n")
				switch incoming.event {
				case "LOGIN":
					fmt.Printf("Server received LOGIN for %q\n", incoming.chara)
					if activeUsers[incoming.chara] == nil {
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
						charToLogIn := chara.ActiveCharacter{ResponseChannel: incoming.returnChannel, Cooldown: 0, CharData: charSheet, IncomingCmds: []string{}}
						activeUsers[incoming.chara] = &charToLogIn
						incoming.returnChannel <- fmt.Sprintf("Welcome to Endless Crystal Blue MUD, %s.\n", incoming.chara)
					} else {
						incoming.returnChannel <- "Character already logged in.\n"
						activeUsers[incoming.chara].ResponseChannel <- "Duplicate login attempt.\n"
						close(incoming.returnChannel)
					}
				case "QUIT":
					if incoming.returnChannel == activeUsers[incoming.chara].ResponseChannel {
						delete(activeUsers, incoming.chara)
						close(incoming.returnChannel)
					} else {
						incoming.returnChannel <- "Received invalid QUIT message.\n"
						activeUsers[incoming.chara].ResponseChannel <- "Received invalid QUIT message.\n"
					}
				default:
					log.Fatalf("Unexpected control message %q\n", incoming.event)
				}
			case incoming := <-servChan:
				fmt.Printf("Received input message on tick %v.\n", tickCounter)
				activeUsers[incoming.chara].IncomingCmds = append(activeUsers[incoming.chara].IncomingCmds, incoming.input)
			default:
			}
		}
		doServerTick(worldRooms, activeUsers)
	}
}
