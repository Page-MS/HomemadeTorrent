package parser

import (
	"testing"
)

func TestEncode(t *testing.T) {
	// minimal case
	msg, err := Decode("ACTION:bijour\nID:bijour_id\n")
	if err != nil {
		t.Errorf("Should not have errored: %s", err)
		return;
	}
	if msg.Action != "bijour" {
		t.Errorf("ACTION value should be 'bijour', found %s", msg.Action)
		return;
	}
	if msg.Id != "bijour_id" {
		t.Errorf("ID value should be 'bijour_id', found %s", msg.Id)
		return;
	}

	// payload case
	msg, err = Decode("ACTION:bijour\nID:bijour_id\nPAYLOAD_LEN:5\nbijour")
	if err != nil {
		t.Errorf("Should not have errored: %s", err)
		return;
	}
	if msg.Action != "bijour" {
		t.Errorf("ACTION value should be 'bijour', found %s", msg.Action)
		return;
	}
	if msg.Id != "bijour_id" {
		t.Errorf("ID value should be 'bijour_id', found %s", msg.Id)
		return;
	}
	if msg.payload_len != 5 {
		t.Errorf("Payload_len value should be '5', found %d", msg.payload_len)
		return;
	}
	if msg.Payload != "bijour" {
		t.Errorf("Payload value should be 'bijour', found %s", msg.Payload)
		return;
	}

	// error case
	msg, err = Decode("ACTIONbijour\nID:bijour_id\nPAYLOAD_LEN:5\nbijour")
	if err == nil {
		t.Errorf("Should have errored but not")
		return;
	}
}
