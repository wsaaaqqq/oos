package main

import (
	"fmt"
	"os"
	"strings"
)

var version = "dev"

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "-v", "--version":
			showVersion()
			return
		case "--upgrade":
			doUpgrade()
			return
		}
	}

	db := dbPath()
	if _, err := os.Stat(db); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Database not found: %s\n", db)
		os.Exit(1)
	}

	initialQuery := strings.Join(os.Args[1:], " ")
	_, err := runTUI(db, initialQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
