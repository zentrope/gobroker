package gobroker

import (
	"bufio"
	"bytes"
	"encoding/binary"
)

type typecode int8

const (
	BrokerPubMessage   typecode = 1
	BrokerSubMessage   typecode = 2
	BrokerUnsubMessage typecode = 3
	BrokerAckMessage   typecode = 4
)

type Message struct {
	Code    typecode
	Topic   string
	Payload []byte
}

func IsPubMessage(b byte) bool {
	return BrokerPubMessage == typecode(b)
}

func IsSubMessage(b byte) bool {
	return BrokerSubMessage == typecode(b)
}

func IsUnsubMessage(b byte) bool {
	return BrokerUnsubMessage == typecode(b)
}

func IsAckMessage(b byte) bool {
	return BrokerAckMessage == typecode(b)
}

func (m Message) IsPubMessage() bool {
	return BrokerPubMessage == m.Code
}

func (m Message) IsSubMessage() bool {
	return BrokerSubMessage == m.Code
}

func (m Message) IsUnsubMessage() bool {
	return BrokerUnsubMessage == m.Code
}

func (m Message) IsAckMessage() bool {
	return BrokerAckMessage == m.Code
}

func (m Message) TypeOf() string {
	switch m.Code {
	case BrokerPubMessage:
		return "PubMessage"
	case BrokerSubMessage:
		return "SubMessage"
	case BrokerUnsubMessage:
		return "UnsubMessage"
	default:
		return "UnknownMessage"
	}
}

func NewPubMessage(topic string, payload []byte) Message {
	return Message{BrokerPubMessage, topic, payload}
}

func NewSubMessage(topic string) Message {
	return Message{BrokerSubMessage, topic, make([]byte, 0)}
}

func NewUnsubMessage(topic string) Message {
	return Message{BrokerUnsubMessage, topic, make([]byte, 0)}
}

func NewAckMessage() Message {
	return Message{BrokerAckMessage, "", make([]byte, 0)}
}

func Encode(message Message) ([]byte, error) {
	return message.Encode()
}

func (message Message) Encode() ([]byte, error) {

	if message.Code == BrokerAckMessage {
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, message.Code)
		return buf.Bytes(), nil
	}

	t := []byte(message.Topic)
	tl := uint8(len(t))

	var data []interface{}

	if message.Code == BrokerPubMessage {
		p := []byte(message.Payload)
		pl := uint16(len(p))
		data = []interface{}{message.Code, tl, t, pl, p}
	} else {
		data = []interface{}{message.Code, tl, t}
	}

	buf := new(bytes.Buffer)

	for _, v := range data {
		err := binary.Write(buf, binary.LittleEndian, v)
		if err != nil {
			return make([]byte, 0), err
		}
	}

	return buf.Bytes(), nil
}

func DecodeMessage(reader *bufio.Reader) (Message, error) {
	var err error
	var code typecode
	var tlen uint8

	err = binary.Read(reader, binary.LittleEndian, &code)
	if err != nil {
		return Message{}, err
	}

	if code == BrokerAckMessage {
		return NewAckMessage(), nil
	}

	err = binary.Read(reader, binary.LittleEndian, &tlen)
	if err != nil {
		return Message{}, err
	}

	topic := make([]byte, tlen)

	err = binary.Read(reader, binary.LittleEndian, &topic)
	if err != nil {
		return Message{}, err
	}

	payload := make([]byte, 0)

	if code == BrokerPubMessage {

		var mlen uint16

		err = binary.Read(reader, binary.LittleEndian, &mlen)
		if err != nil {
			return Message{}, err
		}

		payload = make([]byte, mlen)
		err = binary.Read(reader, binary.LittleEndian, &payload)
		if err != nil {
			return Message{}, err
		}
	}

	return Message{code, string(topic), payload}, nil
}
