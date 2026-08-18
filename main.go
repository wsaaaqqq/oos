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
			tag := ""
			if len(os.Args) >= 3 {
				tag = os.Args[2]
			}
			doUpgrade(tag)
			return
		case "--monitor":
			db := dbPath()
			if _, err := os.Stat(db); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Database not found: %s\n", db)
				os.Exit(1)
			}
			sessionID := ""
			if len(os.Args) >= 3 && !strings.HasPrefix(os.Args[2], "-") {
				sessionID = os.Args[2]
			}
			if err := runMonitor(db, sessionID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
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
