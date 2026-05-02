package snapshot

import "HomemadeTorrent/pkg/registre"

type Color string

const (
	White Color = "blanc"
	Red   Color = "rouge"
)

type Snapshot struct {
	MyColor     Color
	IsInitiator bool
	Bilan       int // émis - reçus (préclic)

	// snapshot
	SavedRegister registre.Registre
	SavedVector   []int // datation

	NbEtatsAttendus int
	NbMsgAttendus   int
	CollectedStates []registre.Registre
}

func (s *Snapshot) SetRed() {
	s.MyColor = Red
}

// Méthode pour vérifier si on doit déclencher la terminaison
func (s *Snapshot) IsReadyToTerminate() bool {
	return s.NbEtatsAttendus == 0 && s.NbMsgAttendus == 0
}
