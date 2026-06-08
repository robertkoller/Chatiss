package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/robertkoller/Chatiss/server/stun"
)

func main() {
	config := stun.DefaultServerConfig()
	server := stun.NewServer()

	if err := server.Start(config); err != nil {
		log.Fatal("Failed to start STUN server:", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	server.Stop()
}
