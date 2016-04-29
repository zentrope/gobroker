package gobroker

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

//-----------------------------------------------------------------------------
// Topic
//-----------------------------------------------------------------------------

type topic struct {
	name           string             // topic name
	lifecycle_ch   chan int           // For terminating the processing loop
	incoming_ch    chan []byte        // Messages to send to subscribers
	subscribe_ch   chan *brokerClient // Subscribe a client to the topic
	unsubscribe_ch chan *brokerClient // Remove client from subscribers
	subscribers    []*brokerClient    // Current subscribers
}

func newTopic(name string) *topic {
	return &topic{
		name:           name,
		incoming_ch:    make(chan []byte),
		subscribe_ch:   make(chan *brokerClient),
		unsubscribe_ch: make(chan *brokerClient),
		subscribers:    make([]*brokerClient, 0),
	}
}

func (topic *topic) process() {
	go func() {
		for {
			select {

			case _ = <-topic.lifecycle_ch:
				log.Printf(" [%s] quitting", topic.name)
				return

			case msg := <-topic.incoming_ch:
				num_subs := len(topic.subscribers)
				if num_subs != 0 {
					log.Println("------------------------------------------------------")
					log.Printf(" [%s] num subscribers: %d", topic.name, num_subs)
					for _, client := range topic.subscribers {
						log.Printf(" [%s] broadcast: [%v]", topic.name, client.conn.RemoteAddr())
						client.publish(msg)
					}
					log.Println("------------------------------------------------------")
				}

			case client := <-topic.subscribe_ch:
				log.Printf(" [%s] add.client: [%v]", topic.name, client.conn.RemoteAddr())
				topic.subscribers = append(topic.subscribers, client)

			case client := <-topic.unsubscribe_ch:
				log.Printf(" [%s] rm.client: [%v]", topic.name, client.conn.RemoteAddr())
				clients := make([]*brokerClient, 0)
				for _, subscriber := range topic.subscribers {
					if client != subscriber {
						clients = append(clients, subscriber)
					}
				}
				topic.subscribers = clients
			}
		}
	}()
}

func (topic *topic) subscribe(client *brokerClient) {
	topic.subscribe_ch <- client
}

func (topic *topic) unsubscribe(client *brokerClient) {
	topic.unsubscribe_ch <- client
}

func (topic *topic) publish(msg []byte) {
	topic.incoming_ch <- msg
}

func (topic *topic) start() {
	topic.lifecycle_ch = make(chan int)
	topic.process()
}

func (topic *topic) stop() {
	close(topic.lifecycle_ch) // Will send default value (0)
}

//-----------------------------------------------------------------------------
// Client
//-----------------------------------------------------------------------------

func (client *brokerClient) read() {
	go func() {
		for {
			message, err := DecodeMessage(client.reader)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Printf("Error: %v\n", err)
				break
			}
			log.Printf("read: %s", message.TypeOf())
			client.incoming_ch <- message
		}
		log.Println(" - read terminated")
		client.stop()
	}()
}

func (client *brokerClient) process() {

	go func() {
		for {
			select {

			case _ = <-client.lifecycle_ch:
				return

			case msg := <-client.incoming_ch:
				client.dispatcher.dispatch(client, msg)
			}
		}
	}()
}

func (client *brokerClient) publish(msg []byte) {
	n, err := client.writer.Write(msg)
	client.writer.Flush()
	if err != nil {
		log.Println("ERROR:", err)
	}
	log.Printf(" -> client %v wrote %d bytes", client.conn.RemoteAddr(), n)
}

func (client *brokerClient) start() {
	log.Println("Starting client.")
	client.reader = bufio.NewReader(client.conn)
	client.writer = bufio.NewWriter(client.conn)
	client.process()
	client.read()
}

func (client *brokerClient) stop() {
	log.Println("Stopping client.")
	close(client.lifecycle_ch)
	client.dispatcher.disconnect(client)
}

type brokerClient struct {
	conn         net.Conn // Socket
	reader       *bufio.Reader
	writer       *bufio.Writer
	dispatcher   *Broker      // Hub
	lifecycle_ch chan int     // Client lifecycle
	incoming_ch  chan Message // Data coming in from the network
}

func newBrokerClient(conn net.Conn, dispatcher *Broker) *brokerClient {
	return &brokerClient{
		conn:         conn,
		dispatcher:   dispatcher,
		lifecycle_ch: make(chan int),
		incoming_ch:  make(chan Message, 0),
	}
}

//-----------------------------------------------------------------------------
// Broker
//-----------------------------------------------------------------------------

func (broker *Broker) accept() {
	go func() {
		for {
			conn, err := broker.listener.Accept()
			if err == nil {
				broker.accept_ch <- conn
			} else {
				if strings.Contains(err.Error(), "closed network") {
					log.Println(" - shutting down listener.")
				} else {
					log.Println(" - listener/accept error ->", err.Error())
				}
				return
			}
		}
	}()
}

func (broker *Broker) dispatch(client *brokerClient, msg Message) {

	switch msg.Code {

	case BrokerPubMessage:
		broker.publish_ch <- pubEvent{client, msg}

	case BrokerSubMessage:
		broker.subscribe_ch <- subEvent{client, msg.Topic}

	case BrokerUnsubMessage:
		log.Println(" - we got an unsubscribe message")

	}
}

func (broker *Broker) disconnect(client *brokerClient) {
	broker.delete_ch <- client
}

func (broker *Broker) processLoop() {

	go func() {
		for {
			select {

			case _ = <-broker.lifecycle_ch:
				break

			case event := <-broker.publish_ch:
				// Publish to a topic if it exists. If it doesn't exist, that
				// means there are no subscribers, so there's no point in
				// sending a message. We'll just drop it.

				name := event.message.Topic
				if topic := broker.topics[name]; topic != nil {
					msg, err := event.message.Encode()
					if err != nil {
						log.Println("Error: unable to encode msg for delivery.", msg)
					} else {
						topic.publish(msg)
					}
				} else {
					log.Printf("Error: unable to find topic [%s] for delivery.", name)
				}

			case event := <-broker.subscribe_ch:
				var name = event.topic
				var topic *topic

				topic, ok := broker.topics[name]
				if !ok {
					topic = newTopic(name)
					topic.start()
					broker.topics[name] = topic
				}
				topic.subscribe(event.client)

				log.Printf("topics: %v", broker.topics)

			case conn := <-broker.accept_ch:
				log.Println(" - incoming connection")
				nc := len(broker.clients)
				client := newBrokerClient(conn, broker)
				broker.clients = append(broker.clients, client)
				client.start()
				log.Printf(" - %d -> %d clients\n", nc, len(broker.clients))

			case client := <-broker.delete_ch:
				log.Println(" - DELETE:", client.conn.RemoteAddr())
				for _, topic := range broker.topics {
					log.Printf(" - unsub from %s", topic.name)
					topic.unsubscribe(client)
				}

				log.Println(" - deleting client from server")
				numClients := len(broker.clients)
				newClients := make([]*brokerClient, 0)

				for _, item := range broker.clients {
					if item == client {
						continue
					}
					newClients = append(newClients, item)
				}

				broker.clients = newClients
				log.Printf(" - %d to %d clients\n", numClients, len(broker.clients))
			}
		}
	}()
}

func (broker *Broker) Start() error {
	log.Println("Starting broker.")
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", broker.port))
	if err != nil {
		return err
	}
	broker.listener = l
	broker.accept()
	broker.processLoop()
	return nil
}

func (broker *Broker) Stop() {
	log.Println("Stopping broker.")
	broker.listener.Close()
}

type pubEvent struct {
	client  *brokerClient
	message Message
}

type subEvent struct {
	client *brokerClient
	topic  string
}

type Broker struct {
	listener     net.Listener
	port         int
	clients      []*brokerClient
	topics       map[string]*topic
	accept_ch    chan net.Conn
	delete_ch    chan *brokerClient
	publish_ch   chan pubEvent
	subscribe_ch chan subEvent
	lifecycle_ch chan struct{}
}

func NewBroker(port int) *Broker {
	return &Broker{
		port:         port,
		clients:      make([]*brokerClient, 0),
		topics:       make(map[string]*topic),
		accept_ch:    make(chan net.Conn),      // incoming client connections
		delete_ch:    make(chan *brokerClient), // dropped client connections
		publish_ch:   make(chan pubEvent),
		subscribe_ch: make(chan subEvent),
	}
}
