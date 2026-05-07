package snapshot

import "HomemadeTorrent/pkg/registre"

type Color string

const (
	White Color = "blanc"
	Red   Color = "rouge"
)

const (
	MARKER          string = "MARKER"
	PREPOST_COLLECT string = "PREPOST_COLLECT"
	STATE_COLLECT   string = "STATE_COLLECT"
	RESET_SNAPSHOT  string = "RESET_SNAPSHOT"
)

type SiteState struct {
	SiteID   string
	Register registre.Registre
	Vector   []int
}
type Snapshot struct {
	MyColor     Color
	IsInitiator bool
	InitiatorID int
	Bilan       int // émis - reçus (préclic)

	// snapshot
	SavedRegister registre.Registre
	SavedVector   []int // datation

	NbEtatsAttendus   int
	NbMsgAttendus     int
	CollectedStates   []SiteState
	CollectedPreposts []string
}

// Structure finale pour le fichier JSON
type GlobalSnapshot struct {
	SnapshotID        string      `json:"snapshot_id"`
	CollectedStates   []SiteState `json:"sites_states"`
	CollectedPreposts []string    `json:"prepost_messages"`
}

func (s *Snapshot) SetRed() {
	s.MyColor = Red
}

// Méthode pour vérifier si on doit déclencher la terminaison
func (s *Snapshot) IsReadyToTerminate() bool {
	return s.NbEtatsAttendus == 0 && s.NbMsgAttendus == 0
}
