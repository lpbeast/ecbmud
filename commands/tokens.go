package commands

import (
	"strings"
)

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL"
	EOL     = "EOL"
	NOOP    = "NOOP"
	QUIT    = "QUIT"

	IDENT  = "IDENT"
	NUMBER = "NUMBER"

	DIRECTION = "DIRECTION"

	GO   = "GO"
	GET  = "GET"
	PUT  = "PUT"
	DROP = "DROP"
	USE  = "USE"
	LOOK = "LOOK"

	SCORE     = "SCORE"
	INVENTORY = "INVENTORY"
	EQUIPMENT = "EQUIPMENT"

	PERIOD    = "."
	DQUOTE    = "\""
	SQUOTE    = "'"
	COLON     = ":"
	SEMICOLON = ";"

	HERE = "HERE"
	ALL  = "ALL"
	ME   = "ME"
	IN   = "IN"
	FROM = "FROM"
)

var keywords = map[string]TokenType{
	"quit": QUIT,

	"north": DIRECTION,
	"south": DIRECTION,
	"east":  DIRECTION,
	"west":  DIRECTION,
	"up":    DIRECTION,
	"down":  DIRECTION,
	"n":     DIRECTION,
	"s":     DIRECTION,
	"e":     DIRECTION,
	"w":     DIRECTION,
	"u":     DIRECTION,
	"d":     DIRECTION,

	"northwest": DIRECTION,
	"southwest": DIRECTION,
	"northeast": DIRECTION,
	"southeast": DIRECTION,
	"nw":        DIRECTION,
	"sw":        DIRECTION,
	"ne":        DIRECTION,
	"se":        DIRECTION,

	"go":   GO,
	"get":  GET,
	"put":  PUT,
	"drop": DROP,
	"use":  USE,
	"look": LOOK,
	"l":    LOOK,

	"score":     SCORE,
	"inventory": INVENTORY,
	"equipment": EQUIPMENT,
	"sc":        SCORE,
	"i":         INVENTORY,
	"inv":       INVENTORY,
	"eq":        EQUIPMENT,
}

var keywordsList = []string{
	"quit",
	"north",
	"south",
	"east",
	"west",
	"up",
	"down",
	"northwest",
	"southwest",
	"northeast",
	"southeast",
	"go",
	"get",
	"put",
	"drop",
	"use",
	"look",
	"score",
	"inventory",
	"equipment",
}

var specialIdents = map[string]TokenType{
	"here": HERE,
	"all":  ALL,
	"self": ME,
	"me":   ME,
	"in":   IN,
	"from": FROM,
}

var dirList = []string{
	"north",
	"south",
	"east",
	"west",
	"up",
	"down",
	"northwest",
	"southwest",
	"northeast",
	"southeast",
}

func lookupCommand(ident string) Token {
	if tok, ok := keywords[ident]; ok {
		return Token{tok, ident}
	} else {
		newIdent := AutoComplete(ident, keywordsList)
		if tok, ok := keywords[newIdent]; ok {
			return Token{tok, ident}
		}
	}
	return Token{ILLEGAL, ident}
}

func lookupIdent(ident string) Token {
	if tok, ok := specialIdents[ident]; ok {
		return Token{tok, ident}
	}
	return Token{IDENT, ident}
}

func AutoComplete(stub string, words []string) string {
	for _, s := range words {
		if strings.HasPrefix(s, stub) {
			return s
		}
	}
	return stub
}

func AutoCompleteDirs(stub string) string {
	switch stub {
	case "nw":
		return "northwest"
	case "ne":
		return "northeast"
	case "se":
		return "southeast"
	case "sw":
		return "southwest"
	default:
		for _, s := range dirList {
			if strings.HasPrefix(s, stub) {
				return s
			}
		}
	}
	return stub
}
