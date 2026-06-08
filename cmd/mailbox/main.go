package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/robertkoller/Chatiss/server/mailbox"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "", "SQLite database path (default: $HOME/chatiss-mailbox.db)")
	flag.Parse()

	if *dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("cannot determine home directory:", err)
		}
		*dbPath = filepath.Join(home, "chatiss-mailbox.db")
	}

	store, err := mailbox.OpenStore(*dbPath)
	if err != nil {
		log.Fatal("failed to open mailbox store:", err)
	}
	defer store.Close()

	srv := mailbox.NewServer(store)
	if err := srv.ListenAndServe(*addr); err != nil {
		log.Fatal(err)
	}
}
