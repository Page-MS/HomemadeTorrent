package parser

import (
	"slices"
	"strings"
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
		t.Errorf("STAMP value should be 123, found %d", msg.Stamp)
		return
	}
	if slices.Compare(msg.Vect, []int{123, 111, 333}) != 0 {
		t.Errorf("VECT value should be [123 111 333], found %v", msg.Vect)
		return
	}

	// payload case
	msg, err = Decode("ACTION:bijour\nID:bijour_id\nPAYLOAD_LEN:6\nbijour\n")
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
	if msg.Payload_len != 6 {
		t.Errorf("Payload_len value should be 6, found %d", msg.Payload_len)
		return
	}
	if msg.Payload != "bijour" {
		t.Errorf("Payload value should be 'bijour', found %s", msg.Payload)
		return
	}

	// color + bilan + object case
	msg, err = Decode(
		"ACTION:test\nID:1\nDEST:0\nSENDER:1\nSTAMP:1\nVECT:1,2\nCOLOR:red\nBILAN:42\nOBJECT:obj",
	)
	if err != nil {
		t.Errorf("Should not have errored: %s", err)
		return
	}

	if msg.Color != "red" {
		t.Errorf("COLOR should be 'red', found %s", msg.Color)
		return
	}
	if msg.Bilan != 42 {
		t.Errorf("BILAN should be 42, found %d", msg.Bilan)
		return
	}
	if msg.Object != "obj" {
		t.Errorf("OBJECT should be 'obj', found %s", msg.Object)
		return
	}

	// error case
	msg, err = Decode("ACTIONbijour\nID:bijour_id\nPAYLOAD_LEN:5\nbijour")
	if err == nil {
		t.Errorf("Should have errored but did not")
		return
	}
}

func TestEncode(t *testing.T) {
	// minimal case
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

	res := "ACTION:bijour\nID:je-suis-un-uuid\nDEST:0\nSENDER:0\nSTAMP:0\nVECT:\n"
	if str != res {
		t.Errorf("Encode should have produced:\n%s\nbut got:\n%s", res, str)
	}

	// error: missing action
	_, err = Encode(Message{
		Action: "",
		Id:     "uuid",
		Chunk:  -1,
		Dest:   "0",
		Sender: "0",
	})
	if err == nil {
		t.Errorf("Should have errored but did not")
		return
	}

	// uuid generation + chunk filtering + payload
	str, err = Encode(Message{
		Action:  "action",
		Chunk:   0,
		Dest:    "0",
		Sender:  "0",
		Payload: "hello",
	})
	if err != nil {
		t.Errorf("Should not have errored: %s", err)
		return
	}

	if !strings.Contains(str, "ACTION:action") {
		t.Errorf("Missing ACTION in output")
	}

	// full message test (all fields)
	str, err = Encode(Message{
		Action: "test",
		Id:     "superId",
		Dest:   "0",
		Sender: "13",
		Vect:   []int{111, 333},
		Stamp:  111,
		Color:  "blue",
		Bilan:  7,
		Object: "obj",
	})
	if err != nil {
		t.Errorf("Should not have errored: %s", err)
		return
	}

	expected := "ACTION:test\nID:superId\nDEST:0\nSENDER:13\nSTAMP:111\nVECT:111,333\nCHUNK:0\nOBJECT:obj\nCOLOR:blue\nBILAN:7\n"
	if str != expected {
		t.Errorf("Encode mismatch:\nEXPECTED:\n%s\nGOT:\n%s", expected, str)
	}
}
