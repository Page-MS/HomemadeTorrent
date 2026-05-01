package snapshot

import (
	"testing"
)

func TestSnapshotStateTransition(t *testing.T) {
	s := &Snapshot{
		MyColor: White,
		Bilan:   0,
	}

	// Test du passage au rouge
	s.SetRed()
	if s.MyColor != Red {
		t.Errorf("Attendu: %s, Obtenu: %s", Red, s.MyColor)
	}

	// Test de la condition de terminaison
	s.NbEtatsAttendus = 0
	s.NbMsgAttendus = 0
	if !s.IsReadyToTerminate() {
		t.Error("Le snapshot devrait être prêt à se terminer quand les compteurs sont à 0")
	}
}
