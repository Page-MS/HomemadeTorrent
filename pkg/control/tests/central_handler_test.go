package control

import (
	"HomemadeTorrent/pkg/control"
	"HomemadeTorrent/pkg/parser"
	"testing"
)

func setupController() *control.Controller {
	siteIDs := []string{"A", "B", "C"}
	return control.NewController("A", siteIDs)
}

func TestHandleIncoming_ClockUpdate(t *testing.T) {
	ctrl := control.NewController("SiteA", []string{"SiteA", "SiteB"})

	msg := parser.Message{
		Id:     "msg1",
		Sender: "SiteB",
		Dest:   "SiteA",
		Action: "UNKNOWN",
		Stamp:  15,
		Vect:   []int{0, 5},
	}

	raw, _ := parser.Encode(msg)

	ctrl.HandleIncomingFromNetwork(raw)

	expectedMin := 16
	if ctrl.Lamport.GetValue() < expectedMin {
		t.Errorf("Lamport non mis à jour. Attendu ≥ %d, obtenu %d",
			expectedMin, ctrl.Lamport.GetValue())
	}
}

func TestHandleIncoming_MessageForSelf(t *testing.T) {
	ctrl := setupController()

	msg := parser.Message{
		Id:     "1",
		Sender: "B",
		Dest:   "A",
		Action: "UNKNOWN",
		Stamp:  1,
	}

	raw, _ := parser.Encode(msg)

	res := ctrl.HandleIncomingFromNetwork(raw)

	// Message traité mais pas forwardé
	if len(res) != 0 {
		t.Errorf("Expected no forward, got %d messages", len(res))
	}
}

func TestHandleIncoming_Broadcast(t *testing.T) {
	ctrl := setupController()

	msg := parser.Message{
		Id:     "2",
		Sender: "B",
		Dest:   control.BROADCAST,
		Action: "UNKNOWN",
		Stamp:  1,
		Vect:   []int{0, 1, 0},
	}

	raw, _ := parser.Encode(msg)

	res := ctrl.HandleIncomingFromNetwork(raw)

	if len(res) != 1 {
		t.Errorf("Expected 1 forwarded message, got %d", len(res))
	}
}

func TestHandleIncoming_ForwardOnly(t *testing.T) {
	ctrl := setupController()

	msg := parser.Message{
		Id:     "3",
		Sender: "B",
		Dest:   "C",
		Action: "UNKNOWN",
		Stamp:  1,
		Vect:   []int{0, 1, 0},
	}

	raw, _ := parser.Encode(msg)

	res := ctrl.HandleIncomingFromNetwork(raw)

	if len(res) != 1 {
		t.Errorf("Expected forward, got %d messages", len(res))
	}
}

func TestHandleIncoming_DuplicateMessage(t *testing.T) {
	ctrl := setupController()

	msg := parser.Message{
		Id:     "4",
		Sender: "B",
		Dest:   control.BROADCAST,
		Action: "UNKNOWN",
		Stamp:  1,
		Vect:   []int{0, 1, 0},
	}

	raw, _ := parser.Encode(msg)

	// première réception
	ctrl.HandleIncomingFromNetwork(raw)

	// deuxième réception
	res := ctrl.HandleIncomingFromNetwork(raw)

	if len(res) != 0 {
		t.Errorf("Expected duplicate to be ignored")
	}
}

func TestHandleIncoming_SelfMessageIgnored(t *testing.T) {
	ctrl := setupController()

	msg := parser.Message{
		Id:     "6",
		Sender: "A",
		Dest:   "B",
		Action: "UNKNOWN",
		Stamp:  1,
		Vect:   []int{1, 0, 0},
	}

	raw, _ := parser.Encode(msg)

	res := ctrl.HandleIncomingFromNetwork(raw)

	if len(res) != 0 {
		t.Errorf("Self message should be ignored")
	}
}
