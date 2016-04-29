package gobroker

import (
	"testing"
)

func TestPubMessage(t *testing.T) {

	m := NewPubMessage("sys.alert", []byte("the.message"))

	e, _ := m.Encode()
	d, _ := Decode(e)

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
