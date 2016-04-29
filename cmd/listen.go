package main

import (
	"github.com/zentrope/gobroker"
	"io"
	"log"
	"os"
)

func handleMessages(client *gobroker.Client) {
	for {
		msg, err := client.Receive()
		if err == io.EOF {
			log.Println("EOF:", err)
			return
		}

		if err != nil {
			log.Println("ERROR:", err)
			return
		}

		log.Printf(" -> msg: [%s] %s\n", msg.Topic, string(msg.Payload))
	}
}

func main() {
	log.Println("Hello Client")

	client := gobroker.NewClient("localhost", 61626, 10)

	log.Println(" - starting client")
	err := client.Start()
	if err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}

	log.Println(" - subscribing")
	err = client.Subscribe("sys.alert")
	if err != nil {
		panic(err)
	}

	log.Println(" - publishing")
	err = client.Publish("sys.alert", []byte("This is an alert"))
	if err != nil {
		panic(err)
	}

	log.Println(" - publishing")
	err = client.Publish("sys.alert", []byte("This is another alert"))
	if err != nil {
		panic(err)
	}

	handleMessages(client)

	log.Println(" - done")
	client.Stop()

}
