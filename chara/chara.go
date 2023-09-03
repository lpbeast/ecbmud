package chara

import (
	"crypto/sha512"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const charListFile = "chara/charlist.csv"

var invalidNames = map[string]string{
	"new":  "new",
	"quit": "quit",
	"look": "look",
	"save": "save",
	"get":  "get",
	"kill": "kill",
	"cast": "cast",
}

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
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_RDONLY, 0600)
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
			nameList, err := getNameList(charListFile)
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
	nameList, err := getNameList(charListFile)
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
	newChara := []string{name, pwHash}

	charFile, err := os.OpenFile(charListFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer charFile.Close()
	w := csv.NewWriter(charFile)
	if err := w.Write(newChara); err != nil {
		log.Fatal(err)
	}
	w.Flush()
	ch <- fmt.Sprintf("Success:%s", name)
}