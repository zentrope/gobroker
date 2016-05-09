package gobroker

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type Client struct {
	name    string
	conn    net.Conn
	reader  *bufio.Reader
	recv_ch chan readResult
	ack_ch  chan bool
	host    string
	port    int
}

type readResult struct {
	message Message
	err     error
}

func NewClient(name, host string, port int, bufsize int) *Client {
	return &Client{
		recv_ch: make(chan readResult, bufsize),
		ack_ch:  make(chan bool, 0),
		host:    host,
		port:    port,
		name:    name,
	}
}

func (c *Client) send(data []byte) error {

	_, err := c.conn.Write(data)

	// fmt.Printf("[%s] wrote %d of %d bytes to socket\n", c.name, n, len(data))
	if err != nil {
		log.Println("write:", err)
		return err
	}

	// fmt.Printf("[%s] waiting on broker.ack()\n", c.name)
	<-c.ack_ch
	// fmt.Printf("[%s] unblocking send\n", c.name)
	return nil
}

func (client *Client) readLoop() {
	go func() {
		for {
			// TODO: Need to distinguish between a network error
			//       and a decoding error.
			message, err := DecodeMessage(client.reader)

			if err != nil {
				// TODO: Remove this once we can distinguish between a
				//       network error and a decoding error.
				if strings.Contains(fmt.Sprintf("%v", err), "closed network") {
					// fmt.Printf("[%s] (read) internal closed\n", client.name)
					break
				}
			}

			if message.IsAckMessage() {
				// fmt.Printf("[%s] (read) got an ack message\n", client.name)
				client.ack_ch <- true
			} else {
				if err != nil {
					if err == io.EOF {
						// fmt.Printf("[%s] (read) EOF\n", client.name)
						break
					}
				}
				// fmt.Printf("[%s] (read) got a regular message [%d %s]\n", client.name,
				//	message.Code,
				//	string(message.Payload))
				client.recv_ch <- readResult{message, err}
			}
		}
		// fmt.Printf("[%s] read-loop exited\n", client.name)
	}()
}

func (c *Client) Receive() (Message, error) {
	// fmt.Printf("[%s] client.recv\n", c.name)
	result := <-c.recv_ch
	return result.message, result.err
}

func (client *Client) Publish(topic string, data []byte) error {
	// fmt.Printf("[%s] client.publish\n", client.name)

	packet, err := NewPubMessage(topic, data).Encode()

	if err != nil {
		return err
	}

	return client.send(packet)
}

func (client *Client) Subscribe(topic string) error {
	// fmt.Printf("[%s] client.subscribe\n", client.name)

	packet, err := NewSubMessage(topic).Encode()
	if err != nil {
		return err
	}

	return client.send(packet)
}

func (c *Client) Unsubscribe(topic string) error {
	// fmt.Printf("[%s] client.unsubscribe\n", c.name)
	packet, err := NewUnsubMessage(topic).Encode()
	if err != nil {
		return err
	}
	return c.send(packet)
}

func (client *Client) Start() error {
	endpoint := fmt.Sprintf("%s:%d", client.host, client.port)
	conn, err := net.Dial("tcp", endpoint)

	if err != nil {
		return err
	}

	client.conn = conn
	client.reader = bufio.NewReader(conn)
	client.readLoop()
	return nil
}

func (c *Client) Stop() error {
	// fmt.Printf("[%s] client.stop\n", c.name)
	return c.conn.Close()
}
