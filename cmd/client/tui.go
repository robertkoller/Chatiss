package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/robertkoller/Chatiss/app"
)

// TUI is what we are working with in the terminal and dictates events
type TUI struct {
	service    *app.Service
	myUsername string

	mu         sync.Mutex
	activePeer string
}

// Definitely does not make a new TUI...
func NewTUI(service *app.Service, username string) *TUI {
	return &TUI{service: service, myUsername: username}
}

// Handles events and calls the appropriate functions
func (t *TUI) HandleEvent(name string, data ...any) {
	switch name {
	case "message:received":
		message, ok := toUIMessage(data)
		if !ok {
			return
		}
		t.mu.Lock()
		active := t.activePeer
		t.mu.Unlock()

		if active == message.From {
			// In conversation mode
			printIncomingMsg(message.From, message.Text, message.Timestamp)
		} else if active != "" {
			// In a different chat
			printEvent(fmt.Sprintf("new message from %s (type /back then /chat %s)", message.From, message.From))
		} else {
			// At the main menu
			printEvent(fmt.Sprintf("new message from %s — type /chat %s to reply", message.From, message.From))
		}

	case "contact:added":
		contact, ok := toStringMap(data)
		if !ok {
			return
		}
		printEvent(fmt.Sprintf("new contact: %s", contact["username"]))

	case "contact:online":
		contact, ok := toStringMap(data)
		if !ok {
			return
		}
		printEvent(fmt.Sprintf("%s is online", contact["username"]))

	case "contact:offline":
		contact, ok := toStringMap(data)
		if !ok {
			return
		}
		printEvent(fmt.Sprintf("%s went offline", contact["username"]))
	}
}

// Run starts the interactive loop. It blocks until the user quits.
func (t *TUI) Run() {
	printBanner(t.myUsername)
	t.showContacts()
	printHelp(false)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		//fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if t.handleTopCommand(line, scanner) {
			return
		}
	}
}

// handleTopCommand processes commands available at the main menu.
// Returns true when the user wants to exit.
func (t *TUI) handleTopCommand(line string, scanner *bufio.Scanner) bool {
	switch {
	case line == "/quit":
		fmt.Println("Bye.")
		return true

	case line == "/contacts":
		t.showContacts()
		printHelp(false)

	case strings.HasPrefix(line, "/add "):
		username := strings.TrimPrefix(line, "/add ")
		username = strings.TrimSpace(username)
		if username == "" {
			printError("Usage: /add <username>")
			return false
		}
		fmt.Printf("Looking up %q…\n", username)
		if err := t.service.AddContact(username); err != nil {
			printError(err.Error())
		} else {
			printEvent(fmt.Sprintf("added %s", username))
			t.showContacts()
		}

	case strings.HasPrefix(line, "/chat "):
		peer := strings.TrimSpace(strings.TrimPrefix(line, "/chat "))
		if peer == "" {
			printError("Usage: /chat <username>")
			return false
		}
		t.chatLoop(peer, scanner)
		fmt.Println(rule())
		t.showContacts()
		printHelp(false)

	default:
		printError(fmt.Sprintf("Unknown command %q. Try /chat <name> or /add <name>.", line))
	}
	return false
}

// chatLoop enters a conversation with peer. Returns when the user types /back or /quit.
func (t *TUI) chatLoop(peer string, scanner *bufio.Scanner) {
	t.mu.Lock()
	t.activePeer = peer
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		t.activePeer = ""
		t.mu.Unlock()
	}()

	fmt.Printf("\n%s%s── %s%s\n", colBold, colAccent, peer, colReset)

	t.service.Connect(peer)

	// Print recent message history
	messages, err := t.service.GetMessages(peer)
	if err != nil {
		fmt.Printf("%s(no history: %v)%s\n", colMuted, err, colReset)
	} else if len(messages) == 0 {
		fmt.Printf("%s(no messages yet)%s\n", colMuted, colReset)
	} else {
		start := 0
		if len(messages) > 20 {
			start = len(messages) - 20
		}
		for _, m := range messages[start:] {
			printHistoryMsg(m.From, t.myUsername, m.Text, m.Outgoing, m.Timestamp)
		}
	}
	printHelp(true)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "/back" {
			return
		}
		if line == "/quit" {
			fmt.Println("Bye.")
			os.Exit(0)
		}
		if strings.HasPrefix(line, "/") {
			printError("Unknown command. Use /back to return or /quit to exit.")
			continue
		}

		if err := t.service.SendMessage(peer, line); err != nil {
			printError(err.Error())
		} else {
			printOutgoingMsg(line, time.Now().Unix())
		}
	}
}

// Loads all of the contacts
func (t *TUI) showContacts() {
	contacts, err := t.service.GetContacts()
	if err != nil {
		printError("Could not load contacts: " + err.Error())
		return
	}
	fmt.Println()
	if len(contacts) == 0 {
		printNoContacts()
	} else {
		for _, c := range contacts {
			printContactRow(c.Username, c.Online)
		}
	}
}

// toUIMessage safely extracts a UIMessage from an emitEvent data slice.
func toUIMessage(data []any) (app.UIMessage, bool) {
	if len(data) == 0 {
		return app.UIMessage{}, false
	}
	m, ok := data[0].(app.UIMessage)
	return m, ok
}

// toStringMap safely extracts a map[string]string from an emitEvent data slice.
func toStringMap(data []any) (map[string]string, bool) {
	if len(data) == 0 {
		return nil, false
	}
	m, ok := data[0].(map[string]string)
	return m, ok
}
