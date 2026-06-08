package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	colReset  = "\033[0m"
	colMuted  = "\033[90m"
	colAccent = "\033[34m"
	colGreen  = "\033[32m"
	colYellow = "\033[33m"
	colRed    = "\033[31m"
	colBold   = "\033[1m"
	width     = 60
)

// clearLine overwrites the current prompt line before printing something.
// Call before any spontaneous output so it doesn't interleave with the cursor.
func clearLine() {
	fmt.Print("\r\033[K")
}

// reprompt reprints the "> " prompt after spontaneous output.
func reprompt() {
	fmt.Print("> ")
}

func rule() string {
	return colMuted + strings.Repeat("─", width) + colReset
}

func printBanner(username string) {
	fmt.Println()
	fmt.Printf("%sChatiss%s  %slogged in as %s%s\n", colBold+colAccent, colReset, colMuted, colBold+username, colReset)
	fmt.Println(rule())
}

func printContactRow(username string, online bool) {
	dot := colMuted + "○" + colReset
	status := colMuted + "offline" + colReset
	if online {
		dot = colGreen + "●" + colReset
		status = colGreen + "online" + colReset
	}
	fmt.Printf("  %s  %-20s  %s\n", dot, username, status)
}

func printNoContacts() {
	fmt.Printf("  %sNo contacts yet — use /add <username>%s\n", colMuted, colReset)
}

func printHelp(inChat bool) {
	fmt.Println(rule())
	if inChat {
		fmt.Printf("%s/back%s  return to contacts   %s/quit%s  exit\n",
			colAccent, colReset, colAccent, colReset)
	} else {
		fmt.Printf("%s/chat <name>%s  open chat   %s/add <name>%s  add contact   %s/quit%s  exit\n",
			colAccent, colReset, colAccent, colReset, colAccent, colReset)
	}
}

func printIncomingMsg(from, text string, ts int64) {
	t := formatTime(ts)
	clearLine()
	fmt.Printf("%s%s%s  %s\n", colMuted, t, colReset, text)
	fmt.Printf("    %s← %s%s\n", colMuted, from, colReset)
	reprompt()
}

func printOutgoingMsg(text string, ts int64) {
	t := formatTime(ts)
	padding := width - len(text) - len(t) - 2
	if padding < 1 {
		padding = 1
	}
	clearLine()
	fmt.Printf("%s%s%s%s%s\n", strings.Repeat(" ", padding), text, colMuted, "  "+t, colReset)
	reprompt()
}

func printHistoryMsg(from, myUsername, text string, outgoing bool, ts int64) {
	t := formatTime(ts)
	if outgoing {
		padding := width - len(text) - len(t) - 2
		if padding < 1 {
			padding = 1
		}
		fmt.Printf("%s%s%s%s%s\n", strings.Repeat(" ", padding), text, colMuted, "  "+t, colReset)
	} else {
		fmt.Printf("%s%s%s  %s\n", colMuted, t, colReset, text)
	}
	_ = myUsername
}

func printEvent(msg string) {
	clearLine()
	fmt.Printf("%s-- %s --%s\n", colMuted, msg, colReset)
	reprompt()
}

func printNotification(from, text string) {
	clearLine()
	fmt.Printf("%s[%s]%s %s\n", colYellow, from, colReset, text)
	reprompt()
}

func printError(msg string) {
	fmt.Printf("%s%s%s\n", colRed, msg, colReset)
}

func formatTime(ts int64) string {
	t := time.Unix(ts, 0)
	now := time.Now()
	if t.Format("2006-01-02") == now.Format("2006-01-02") {
		return t.Format("15:04")
	}
	return t.Format("Jan 2 15:04")
}
