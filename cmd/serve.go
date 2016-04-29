package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/zentrope/gobroker"
)

//-----------------------------------------------------------------------------
// Main
//-----------------------------------------------------------------------------

// Hook the shutdown signal and run a function
func hookShutdown(fn func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	fn()
}

func lock() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
}

func initialize() {
	// log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Hello Go Mbus")
}

func main() {
	initialize()

	server := gobroker.NewBroker(61626)
	server.Start()

	lock()

	server.Stop()
	time.Sleep(500 * time.Millisecond)
	log.Println("System halt.")
}
