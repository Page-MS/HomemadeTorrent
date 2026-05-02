package parser

import (
	"slices"
	"testing"
)

func TestDecode(t *testing.T) {
	// minimal case
	msg, err := Decode("ACTION:bijour\nID:bijour_id\nDEST:123\nSENDER:1\nSTAMP:123\nVECT:123,111,333")
	if err != nil {
		t.Errorf("Should not have errored: %s", err)
		return
	}
	if msg.Action != "bijour" {
		t.Errorf("ACTION value should be 'bijour', found %s", msg.Action)
		return
	}
	if msg.Id != "bijour_id" {
		t.Errorf("ID value should be 'bijour_id', found %s", msg.Id)
		return
	}
	if msg.Dest != "123" {
		t.Errorf("DEST value should be '123', found %s", msg.Dest)
		return
	}
	if msg.Sender != "1" {
		t.Errorf("SENDER value should be '1', found %s", msg.Sender)
		return
	}
	if msg.Stamp != 123 {
		t.Errorf("STAMP value should be '123', found %d", msg.Stamp)
		return
	}
	if slices.Compare(msg.Vect, []int{123, 111, 333}) != 0 {
		t.Errorf("STAMP value should be [123 111 333], found %d", msg.Vect)
		return
	}

	// payload case
	msg, err = Decode("ACTION:bijour\nID:bijour_id\nPAYLOAD_LEN:5\nbijour")
	if err != nil {
		t.Errorf("Should not have errored: %s", err)
		return
	}
	if msg.Action != "bijour" {
		t.Errorf("ACTION value should be 'bijour', found %s", msg.Action)
		return
	}
	if msg.Id != "bijour_id" {
		t.Errorf("ID value should be 'bijour_id', found %s", msg.Id)
		return
	}
	if msg.payload_len != 5 {
		t.Errorf("Payload_len value should be '5', found %d", msg.payload_len)
		return
	}
	if msg.Payload != "bijour" {
		t.Errorf("Payload value should be 'bijour', found %s", msg.Payload)
		return
	}

	// error case
	msg, err = Decode("ACTIONbijour\nID:bijour_id\nPAYLOAD_LEN:5\nbijour")
	if err == nil {
		t.Errorf("Should have errored but not")
		return
	}
}

func TestEncode(t *testing.T) {
	// no action/chunk/payload
	str, err := Encode(Message{
		Action: "bijour",
		Id:     "je-suis-un-uuid",
		Chunk:  -1,
		Dest:   "0",
		Sender: "0",
	})
	if err != nil {
		t.Errorf("Should not have errored: %s", err)
		return
	}
	res := "ACTION:bijour\nID:je-suis-un-uuid\nDEST:0\nSENDER:0\nSTAMP:0\nVECT:"
	if str != res {
		t.Errorf("Encode operation should have produced '%s', but found %s", res, str)
	}

	// error
	str, err = Encode(Message{
		Action: "",
		Id:     "je-suis-un-uuid",
		Chunk:  -1,
		Dest:   "0",
		Sender: "0",
	})
	if err == nil { // missing action
		t.Errorf("Should have errored but not")
		return
	}

	// generating uuid
	str, err = Encode(Message{
		Action: "action",
		Chunk:  -1,
		Dest:   "0",
		Sender: "0",
	})
	if err != nil { // missing action
		t.Errorf("Should not have errored, but found %s", err)
		return
	}

	// encode stamp & vecto
	str, err = Encode(Message{
		Action: "test",
		Id:     "superId",
		Dest:   "0",
		Sender: "13",
		Vect:   []int{111, 333},
		Stamp:  111,
	})
	if err != nil { // missing action
		t.Errorf("Should not have errored, but found %s", err)
		return
	}
	res = "ACTION:test\nID:superId\nDEST:0\nSENDER:13\nSTAMP:111\nVECT:111,333\nCHUNK:0"
	if str != res {
		t.Errorf("Encode operation should have produced '%s'\n, but found %s", res, str)
	}
}
