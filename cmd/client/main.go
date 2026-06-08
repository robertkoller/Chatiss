package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertkoller/Chatiss/app"
)

func main() {
	passphrase := prompt("Passphrase: ")
	username := prompt("Username:   ")

	// Redirect logs to a file so they don't flood the TUI.
	// The file is truncated on each run so it only contains the current session.
	logPath := logFilePath()
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600); err == nil {
		log.SetOutput(f)
		defer f.Close()
		fmt.Printf("Logs → %s\n", logPath)
	} else {
		log.SetOutput(os.Stderr)
	}

	fmt.Println("Connecting…")

	// tui is set before any service events fire in practice (the first event
	// requires a network round-trip), but the HandleEvent nil-check in the
	// closure below handles the rare race cleanly.
	var tui *TUI

	svc, err := app.NewService(passphrase, username, func(name string, data ...any) {
		if tui != nil {
			tui.HandleEvent(name, data...)
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	defer svc.Stop()

	tui = NewTUI(svc, username)
	tui.Run()
}

func prompt(label string) string {
	fmt.Print(label)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func logFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "/tmp/chatiss-client.log"
	}
	return filepath.Join(dir, "Chatiss", "client.log")
}
