package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	db := dbPath()
	if _, err := os.Stat(db); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Database not found: %s\n", db)
		os.Exit(1)
	}

	initialQuery := strings.Join(os.Args[1:], " ")

	session, err := runTUI(db, initialQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if session == nil {
		return
	}

	if err := openSession(*session); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening session: %v\n", err)
		os.Exit(1)
	}
}
