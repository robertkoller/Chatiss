package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/robertkoller/Chatiss/server/turn"
)

func main() {
	publicIP := "178.128.151.84"
	if len(os.Args) > 1 {
		publicIP = os.Args[1]
	}

	server := turn.NewServer(publicIP)
	if err := server.Start(":13479", ":443"); err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("TURN server shutting down.")
}
