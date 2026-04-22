package control

import "testing"

func TestGetNewDistributedFile(t *testing.T) {
	n := 10
	siteIndex := 5

	df := GetNewDistributedFile(n, siteIndex)

	// Vérifier que l'objet n'est pas nil
	if df == nil {
		t.Fatal("DistributedFile should not be nil")
	}

	// Vérifier le SiteIndex
	if df.SiteIndex != siteIndex {
		t.Errorf("Expected SiteIndex %d, got %d", siteIndex, df.SiteIndex)
	}

	// Vérifier la taille du tableau
	if len(df.Tab) != n {
		t.Errorf("Expected Tab length %d, got %d", n, len(df.Tab))
	}

	// Vérifier l'initialisation des entrées
	for i, entry := range df.Tab {
		if entry.Type != SC_LIBERATION {
			t.Errorf("Tab[%d]: expected Type SC_LIBERATION, got %s", i, entry.Type)
		}
		if entry.Date != 0 {
			t.Errorf("Tab[%d]: expected Date 0, got %d", i, entry.Date)
		}
	}

	// Vérifier l'horloge Lamport initiale (supposée à 0)
	if df.EstampClock.GetValue() != 0 {
		t.Errorf("Expected Lamport clock to be 0, got %d", df.EstampClock.GetValue())
	}
}

func TestSCRequestFromBaseApp(t *testing.T) {
	df := GetNewDistributedFile(10, 5)
	msg := df.SCRequestFromBaseApp()

	// Verifier que c'est une requete de debut de section critique
	if msg.Type != SC_REQUEST {
		t.Errorf("Expected type %s, got %s", SC_REQUEST, msg.Type)
	}

	// Verifier que l'horloge vaut 1 après une action
	if msg.ClockValue != 1 {
		t.Errorf("Expected clock with value 1, got %d", msg.ClockValue)
	}

	// Verifier que c'est envoyé en broadcast
	if msg.IndexDest != -1 {
		t.Errorf("Expected destination index -1, got %d", msg.IndexDest)
	}
}

func TestSCStopFromBaseApp(t *testing.T) {
	df := GetNewDistributedFile(10, 5)
	msg := df.SCStopFromBaseApp()

	// Verifier que c'est une requete de fin de section critique
	if msg.Type != SC_LIBERATION {
		t.Errorf("Expected type %s, got %s", SC_REQUEST, msg.Type)
	}

	// Verifier que l'horloge vaut 1 après une action
	if msg.ClockValue != 1 {
		t.Errorf("Expected clock with value 1, got %d", msg.ClockValue)
	}

	// Verifier que c'est envoyé en broadcast
	if msg.IndexDest != -1 {
		t.Errorf("Expected destination index -1, got %d", msg.IndexDest)
	}
}

func TestSCRequestFromNetwork(t *testing.T) {
	n := 3
	siteIndex := 0

	df := GetNewDistributedFile(n, siteIndex)
	for i := range n {
		df.Tab[i].Date = 10
	}

	// Simuler que le site courant a déjà fait une requête
	df.Tab[siteIndex] = TabEntry{
		Type: SC_REQUEST,
		Date: 3,
	}

	// Message entrant d'un autre site avec une date plus grande
	msg := Message{
		Type:        SC_REQUEST,
		IndexSender: 1,
		ClockValue:  5,
	}

	ack, sc := df.SCRequestFromNetwork(msg)

	// --- Vérification ACK ---
	if ack.Type != ACK {
		t.Errorf("Expected ACK, got %s", ack.Type)
	}

	if ack.IndexSender != siteIndex {
		t.Errorf("ACK sender incorrect, expected %d got %d", siteIndex, ack.IndexSender)
	}

	if ack.IndexDest != msg.IndexSender {
		t.Errorf("ACK dest incorrect, expected %d got %d", msg.IndexSender, ack.IndexDest)
	}

	// --- Vérification mise à jour Tab ---
	entry := df.Tab[msg.IndexSender]
	if entry.Type != SC_REQUEST {
		t.Errorf("Tab not updated correctly, expected SC_REQUEST got %s", entry.Type)
	}

	if entry.Date != msg.ClockValue {
		t.Errorf("Wrong date in Tab, expected %d got %d", msg.ClockValue, entry.Date)
	}

	// --- Vérification SC ---
	// Ici notre requête est plus ancienne (3 < 5) → doit passer
	if !sc {
		t.Errorf("Expected SC access (true), got false")
	}

	// Cas inverse
	msg = Message{
		Type:        SC_REQUEST,
		IndexSender: 1,
		ClockValue:  2,
	}

	_, sc = df.SCRequestFromNetwork(msg)

	if sc {
		t.Errorf("Expected SC access to be false, got true")
	}
}

func TestSCStopFromNetwork(t *testing.T) {
	n := 5
	siteIndex := 0

	df := GetNewDistributedFile(n, siteIndex)
	for i := range n {
		df.Tab[i].Date = 10
	}

	msg := Message{
		Type:        SC_LIBERATION,
		IndexSender: 1,
		ClockValue:  4,
	}

	// Pas de requette donc pas de section critique
	sc := df.SCStopFromNetwork(msg)
	if sc {
		t.Errorf("Expected SC access to be false, got true")
	}

	df.Tab[siteIndex] = TabEntry{
		Type: SC_REQUEST,
		Date: 3,
	}
	df.Tab[msg.IndexSender] = TabEntry{
		Type: SC_LIBERATION,
		Date: 10,
	}

	// Requette de sc en attente donc debut de sc
	sc = df.SCStopFromNetwork(msg)
	if !sc {
		t.Errorf("Expected SC access to be true, got false")
	}
}

func TestAckFromNetwork(t *testing.T) {
	n := 5
	siteIndex := 0

	df := GetNewDistributedFile(n, siteIndex)
	for i := range n {
		df.Tab[i].Date = 10
	}

	msg := Message{
		Type:        ACK,
		IndexSender: 1,
		ClockValue:  4,
	}

	// Pas de requette donc pas de section critique
	sc := df.AckFromNetwork(msg)
	if sc {
		t.Errorf("Expected SC access to be false, got true")
	}

	df.Tab[siteIndex] = TabEntry{
		Type: SC_REQUEST,
		Date: 3,
	}
	df.Tab[msg.IndexSender] = TabEntry{
		Type: SC_LIBERATION,
		Date: 10,
	}

	// Requette de sc en attente donc debut de sc
	sc = df.AckFromNetwork(msg)
	if !sc {
		t.Errorf("Expected SC access to be true, got false")
	}
}
