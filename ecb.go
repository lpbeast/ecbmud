package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
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
	io.WriteString(c, "Welcome to Eternal Crystal Blue MUD\n")
	io.WriteString(c, "Enter a character name\n")
	ch := make(chan string, 20)
	quit := make(chan string)
	sc := bufio.NewScanner(c)
	doLoop := true
	sc.Scan()
	name := sc.Text()

	eventMsg := ctrlMsg{name, "LOGIN", ch}
	ctrlChan <- eventMsg

	go func() {
		for sc.Scan() {
			line := sc.Text()
			if line == "quit" {
				eventMsg := ctrlMsg{name, "QUIT", ch}
				ctrlChan <- eventMsg
				quit <- "quit"
				break
			}
			fmt.Printf("Handler: Sending %q from %q\n", line, name)
			msgForServer := inputMsg{name, line}
			servChan <- msgForServer
		}
	}()

	for doLoop {
		select {
		case <-quit:
			doLoop = false
		case response := <-ch:
			io.WriteString(c, response)
		default:
		}
	}
	c.Close()
}

func main() {
	runServer := true
	servChan := make(chan inputMsg, 20)
	connChan := make(chan net.Conn, 20)
	ctrlChan := make(chan ctrlMsg, 20)
	activeUsers := make(map[string]chan string)

	l, err := net.Listen("tcp", ":4040")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	go func(connChan chan net.Conn) {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}
			connChan <- conn
		}
	}(connChan)

	for runServer {
		select {
		case conn := <-connChan:
			go createConnection(conn, servChan, ctrlChan)
		case incoming := <-ctrlChan:
			switch incoming.event {
			case "LOGIN":
				if activeUsers[incoming.chara] == nil {
					activeUsers[incoming.chara] = incoming.returnChannel
				} else {
					incoming.returnChannel <- "Character already logged in.\n"
					activeUsers[incoming.chara] <- "Duplicate login attempt.\n"
				}
			case "QUIT":
				if incoming.returnChannel == activeUsers[incoming.chara] {
					delete(activeUsers, incoming.chara)
				} else {
					incoming.returnChannel <- "Received invalid QUIT message.\n"
					activeUsers[incoming.chara] <- "Received invalid QUIT message.\n"
				}
			default:
				log.Fatal("Unexpected control message\n")
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
