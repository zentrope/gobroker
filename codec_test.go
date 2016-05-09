package gobroker

import (
	"bufio"
	"bytes"
	"testing"
)

func TestPubMessage(t *testing.T) {

	m := NewPubMessage("sys.alert", []byte("the.message"))

	e, _ := m.Encode()

	r := bufio.NewReader(bytes.NewReader(e))

	d, _ := DecodeMessage(r)

	if m.Topic != d.Topic {
		t.Error("expected matching topic")
	}

	if string(m.Payload) != string(d.Payload) {
		t.Error("expected matching payload")
	}

	if !m.IsPubMessage() {
		t.Error("expected PUB code, got", m.Code)
	}
}

func TestAckMessage(t *testing.T) {
	m := NewAckMessage()

	e, _ := m.Encode()

	r := bufio.NewReader(bytes.NewReader(e))

	d, _ := DecodeMessage(r)

	if !d.IsAckMessage() {
		t.Error("exected ack message")
	}
}
