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
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: oos --monitor <session-id>")
				os.Exit(1)
			}
			db := dbPath()
			if _, err := os.Stat(db); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Database not found: %s\n", db)
				os.Exit(1)
			}
			if err := runMonitor(db, os.Args[2]); err != nil {
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
