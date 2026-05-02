package control

import (
	"HomemadeTorrent/pkg/registre"
	"HomemadeTorrent/pkg/snapshot"
	"testing"
)

func TestSnapshotWorkflow(t *testing.T) {
	// Initialisation
	allSites := []string{"Site0", "Site1", "Site2"}
	c := NewController("Site0", allSites)

	if c.Reg == nil {
		c.Reg = &registre.Registre{}
	}
	// Simuler un état avant le snapshot
	c.Vector.Tick()      // L'horloge passe à [1, 0, 0]
	c.Snapshot.Bilan = 2 // On simule 2 messages envoyés non encore reçus

	// Test de triggerLocalSnapshot (Le Clic)
	c.triggerLocalSnapshot(true) // Site0 est l'initiateur

	if c.Snapshot.MyColor != snapshot.Red {
		t.Errorf("Le site devrait être ROUGE, obtenu: %v", c.Snapshot.MyColor)
	}

	// Vérification du compatge
	// N-1 = 2 sites attendus
	if c.Snapshot.NbEtatsAttendus != 2 {
		t.Errorf("Attendu 2 états, obtenu %d", c.Snapshot.NbEtatsAttendus)
	}
	// Le NbMsgAttendus doit copier le bilan au moment du clic
	if c.Snapshot.NbMsgAttendus != 2 {
		t.Errorf("Attendu 2 messages (bilan), obtenu %d", c.Snapshot.NbMsgAttendus)
	}

	// Test de la datation vectorielle
	if len(c.Snapshot.SavedVector) != 3 {
		t.Error("Le vecteur sauvegardé n'a pas la bonne taille")
	}

	// Test de FinalizeSnapshot
	c.finalizeSnapshot()

	if c.Snapshot.MyColor != snapshot.White {
		t.Error("Le site devrait être revenu à BLANC après la finalisation")
	}

	if c.Snapshot.IsInitiator {
		t.Error("IsInitiator devrait être false après finalisation")
	}
}
