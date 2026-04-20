package clock

import (
	"reflect"
	"testing"
)

func TestVectorClock(t *testing.T) {
	// On simule un réseau de 3 sites, on est le 1
	vc := NewVectorClock(3, 1)

	// Test état initial
	if !reflect.DeepEqual(vc.GetCopy(), []int{0, 0, 0}) {
		t.Errorf("Initialisation ratée: %v", vc.GetCopy())
	}

	// Test Tick (on fait une action)
	vc.Tick()
	expectedAfterTick := []int{0, 1, 0}
	if !reflect.DeepEqual(vc.GetCopy(), expectedAfterTick) {
		t.Errorf("Après Tick: attendu %v, reçu %v", expectedAfterTick, vc.GetCopy())
	}

	// Test Update (on reçoit un message du Site 0 qui était à [2, 0, 0])
	remoteVector := []int{2, 0, 0}
	vc.Update(remoteVector)
	expectedAfterUpdate := []int{2, 2, 0}
	if !reflect.DeepEqual(vc.GetCopy(), expectedAfterUpdate) {
		t.Errorf("Après Update: attendu %v, reçu %v", expectedAfterUpdate, vc.GetCopy())
	}
}
