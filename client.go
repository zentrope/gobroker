package gobroker

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

type Client struct {
	conn    net.Conn
	writer  *bufio.Writer
	reader  *bufio.Reader
	recv_ch chan readResult
	host    string
	port    int
}

type readResult struct {
	message Message
	err     error
}

func NewClient(host string, port int, bufsize int) *Client {
	return &Client{
		recv_ch: make(chan readResult, bufsize),
		host:    host,
		port:    port,
	}
}

func (client *Client) send(data []byte) {
	client.writer.Write(data)
	client.writer.Flush()
}

func (client *Client) readLoop() {
	go func() {
		for {
			message, err := DecodeMessage(client.reader)
			client.recv_ch <- readResult{message, err}
			if err != nil {
				if err == io.EOF {
					break
				}
			}
		}
	}()
}

func (client *Client) Receive() (Message, error) {
	result := <-client.recv_ch
	return result.message, result.err
}

func (client *Client) Publish(topic string, data []byte) error {

	packet, err := NewPubMessage(topic, data).Encode()

	if err != nil {
		return err
	}

	client.send(packet)
	return nil
}

func (client *Client) Subscribe(topic string) error {
	packet, err := NewSubMessage(topic).Encode()
	if err != nil {
		return err
	}

	client.send(packet)
	return nil
}

func (client *Client) Unsubscribe(topic string) error {
	packet, err := NewUnsubMessage(topic).Encode()
	if err != nil {
		return err
	}
	client.send(packet)
	return nil
}

func (client *Client) Start() error {
	endpoint := fmt.Sprintf("%s:%d", client.host, client.port)
	conn, err := net.Dial("tcp", endpoint)

	if err != nil {
		return err
	}

	client.conn = conn
	client.writer = bufio.NewWriter(conn)
	client.reader = bufio.NewReader(conn)

	client.readLoop()
	return nil
}

func (client *Client) Stop() error {
	return client.conn.Close()
}
