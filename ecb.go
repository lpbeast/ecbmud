package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/lpbeast/ecbmud/chara"
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

func createConnection(c net.Conn, servChan chan inputMsg, ctrlChan chan ctrlMsg) {
	ch := make(chan string)
	ic := make(chan string)
	loginChan := make(chan string)
	sc := bufio.NewScanner(c)
	loggedIn := false
	name := ""

	io.WriteString(c, "Welcome to Endless Crystal Blue MUD\n")

	go chara.DoLogin(ch, loginChan)

	go func(inputChan chan string) {
		for sc.Scan() {
			line := sc.Text()
			inputChan <- line
		}
	}(ic)

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
				msgForServer := inputMsg{name, input}
				servChan <- msgForServer
			}
		case resp := <-ch:
			io.WriteString(c, resp)
		default:
		}
	}

	c.Close()
}

func main() {
	runServer := true
	// servChan is for the connection handlers to send user input to the main server
	servChan := make(chan inputMsg)
	// the goroutine that listens for new connections uses connChan to tell the server
	// that there's a new connection, and to hand the connection over to the connection handler
	connChan := make(chan net.Conn)
	// ctrlChan is for the connection handlers to send control messages like LOGIN and QUIT
	// to the main server
	ctrlChan := make(chan ctrlMsg)
	activeUsers := make(map[string]chan string)

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

	fmt.Printf("Server starting.\n")
	for runServer {
		select {
		case conn := <-connChan:
			go createConnection(conn, servChan, ctrlChan)
		case incoming := <-ctrlChan:
			switch incoming.event {
			case "LOGIN":
				fmt.Printf("Server received LOGIN for %q\n", incoming.chara)
				if activeUsers[incoming.chara] == nil {
					activeUsers[incoming.chara] = incoming.returnChannel
					incoming.returnChannel <- fmt.Sprintf("Welcome to Endless Crystal Blue MUD, %s.\n", incoming.chara)
				} else {
					incoming.returnChannel <- "Character already logged in.\n"
					activeUsers[incoming.chara] <- "Duplicate login attempt.\n"
					close(incoming.returnChannel)
				}
			case "QUIT":
				if incoming.returnChannel == activeUsers[incoming.chara] {
					delete(activeUsers, incoming.chara)
					close(incoming.returnChannel)
				} else {
					incoming.returnChannel <- "Received invalid QUIT message.\n"
					activeUsers[incoming.chara] <- "Received invalid QUIT message.\n"
				}
			default:
				log.Fatalf("Unexpected control message %q\n", incoming.event)
			}
		case incoming := <-servChan:
			response := fmt.Sprintf("Server: Received %q from %q\n", incoming.input, incoming.chara)
			fmt.Print(response)
			activeUsers[incoming.chara] <- response
		default:
			if rand.Intn(100) == 0 {
				for _, v := range activeUsers {
					v <- "Random asynchronous event!\n"
					break
				}
			}
			time.Sleep(100 * time.Millisecond)

		}
	}
}
