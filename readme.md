# gobroker

Fun and games to learn a bit of [Go](http://golang.org) programming.

## Rationale

You want to send data to one or more processes running in your system, but
don't know want to have to configure a route to them. You want to
consume data from processes that don't or can't know about your
existence due to policy or developmental time shifting. The usual
broker thing.

Some choices:

 - As a client, you send a payload (bunch of bytes, could be JSON or
   EDN or protobuf) to a topic.

 - As a client, you subscribe to a topic, and get that payload. For
   now, there's no indication of the payload's type. You just have to
   know.

 - Unfancy binary message format.

 - Topics exist as long as there's a subscriber.

 - Messages published to topics with no subscribers are dropped (no
   persistence, durability, etc).

 - Subscriptions live only as long as a connection (no durability,
   etc).

 - TODO: Add a type-hint byte to the protocol.

 - TODO: Eventually compress the payload (because why not)?

 - TODO: Use the ":61626" style address stuff, same as the net
   package.

## Status

The code works for what little hand-testing I'm able to do. Maybe it's
basically done (though perhaps too synchronous).

The broker is embeddable, so I'd like to work out a testing strategy,
which, I think, will teach me a lot more about Go.

## Install

I wouldn't recommend it, but:

```sh
$ go get github.com/zentrope/gobroker
```

**Server**

Set the port and start it. Broker doesn't block the main thread, so
you've got to do that yourself. The idea is that you might run the
Broker embedded in an app with several other concerns.

```go
import (
  "os"
  "os/signal"

  "github.com/zentrope/gobroker"
)

func lock() {
  sigChan := make(chan os.Signal, 1)
  signal.Notify(sigChan, os.Interrupt)
  <-sigChan
}

func main() {
  server := gobroker.NewBroker(61626)
  server.Start()

  lock()
  server.Stop()
}
```

**Client**

The client buffers incoming messages (you can set the buffer
size). You just have to `Receive()` them in a goroutine.

```go
import (
  "log"
  "github.com/zentrope/gobroker"
)

func handle(client *gobroker.Client) {
  for {
    msg, err = client.Receive()
    if err == io.EOF {
      return
    }
    log.Println("msg:", msg)
  }
}

func main() {

  topic := "sys.alert"

  client := gobroker.NewClient("localhost", 61626, 10)
  client.Start()

  client.Subscribe(topic)

  client.Publish(topic, []byte("It's broke."))
  client.Publish(topic, []byte("It's broker."))

  handle(client)

  client.Stop()
}
```

## Running

Something like:

```sh
$ cd gobroker
$ go run cmd/serve.go  # runs the server
$ go run cmd/listen.go # hand testing client
```

## Protocol

**Subscribe**

Something like:

```
int8      : 2 -> Subscribe
uint8     : topic name's size
uint8[]   : topic's name
```

**Unsubscribe**

```
int8      : 3 -> Unsubscribe
uint8     : topic name's size
uint8[]   : topic's name
```

**Publish**

No headers or additional metadata about the message: just push it
through.

```
int8      : 1 = publish
uint8     : topic name's size
uint8[]   : topic's name
uint16    : payload's size
uint8[]   : payload
```

**Implications:**

- Payloads, max size 64k
- Topics, max size 255b

**Compression?**

Would have to wrap the above messages in something that allows
routing without having to decompress. So, maybe only the payload gets
compressed.

The broker pulls out the type (pub, sub, unsub) and topic name. For
pub messages, just pass through the encryption. But the topic name and
type is needed for routing, creating topics, etc.

```
int8      : 1 = publish
uint8     : topic name size
[uint8 ]  : topic name
uint16    : payload size
[uint8 ]  : payload <-- compress
```
