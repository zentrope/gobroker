package gobroker

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"
)

const port = 61626

func TestSingleMessage(t *testing.T) {

	log.SetOutput(ioutil.Discard)

	broker := NewBroker(port)
	broker.Start()

	defer broker.Stop()

	testTopic := "test.alert"

	client := NewClient("zebra", "localhost", port, 0)
	client.Start()
	defer client.Stop()
	defer client.Unsubscribe(testTopic)

	client.Subscribe(testTopic)
	client.Publish(testTopic, []byte("test"))

	msg, _ := client.Receive()

	if msg.Topic != testTopic {
		t.Error("Topic doesn't match subscription.")
	}

	payload := string(msg.Payload)
	if payload != "test" {
		t.Errorf("Payload: expected '%s' but got '%s'", "test", payload)
	}
}

func testMultipleSubscribersSameMessage(t *testing.T) {

	log.SetOutput(ioutil.Discard)

	broker := NewBroker(port)
	broker.Start()
	defer broker.Stop()

	testTopic := "test.alert"
	testPayload := "test.data"

	// sender
	sender := NewClient("sender", "localhost", port, 0)
	sender.Start()

	defer sender.Stop()

	numClients := 2
	var clients []*Client
	for n := 0; n < numClients; n++ {

		c := NewClient(fmt.Sprintf("client-%d", n), "localhost", port, 0)
		c.Start()

		defer c.Stop()
		defer c.Unsubscribe(testTopic)

		// It's possible for a Subscribe action to NOT block
		// before it occurs on the server, so the Receive
		// down below will wait forever.
		err := c.Subscribe(testTopic)
		if err != nil {
			t.Error("Unable to subscribe:", err)
		}

		clients = append(clients, c)
	}

	// time.Sleep(1 * time.Second)

	sender.Publish(testTopic, []byte(testPayload))

	for n := 0; n < numClients; n++ {

		c := clients[n]
		msg, err := c.Receive()

		if err != nil {
			t.Error("Error on receive:", err)
		}

		if msg.Topic != "test.alert" {
			t.Error("Topic doesn't match subscription.")
		}

		payload := string(msg.Payload)
		if payload != testPayload {
			t.Errorf("Payload: expected '%s' but got '%s'", testPayload, payload)
		}
	}

}
