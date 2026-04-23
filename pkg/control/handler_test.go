package control

import (
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
	"strings"
	"testing"
)

func TestHandleIncoming_ClockUpdate(t *testing.T) {
	// Initialisation
	ctrl := NewController(3, 0, "SiteA")
	ctrl.Reg = &registre.Registre{} // On mock le registre pour éviter les crashs

	// simulation d'un message avec une horloge plus grande (15)
	// horloge locale est à 0.
	raw := "ACTION:ACK\nID:Site1\nSTAMP:15\nVECTOR:0,0,0"

	_, _ = ctrl.HandleIncoming(raw)

	expected := 16
	if ctrl.Lamport.GetValue() < expected {
		t.Errorf("L'horloge de Lamport n'a pas été mise à jour. Attendu: %d, Obtenu: %d", expected, ctrl.Lamport.GetValue())
	}
}

func TestProcessDistributedFile_ACK(t *testing.T) {
	ctrl := NewController(3, 1, "Site1")
	ctrl.Reg = &registre.Registre{}

	// requête SC_REQUEST du SiteA
	raw := "ACTION:SC_REQUEST\nID:Site0\nSTAMP:5\nVECTOR:0,0,0"

	response, toBroadcast := ctrl.HandleIncoming(raw)

	// handler doit générer un ACK
	if !strings.Contains(response, "ACTION:ACK") {
		t.Errorf("Le handler aurait dû générer un ACK. Réponse obtenue: %s", response)
	}

	// ACK envoyé au demandeur pas en broadcast
	if toBroadcast {
		t.Errorf("Un ACK ne devrait pas être broadcasté")
	}
}

func TestFullCycle_ReadyToSC(t *testing.T) {
	// 2 sites : SiteA (local), SiteB (distant)
	ctrl := NewController(2, 0, "SiteA")
	ctrl.Reg = &registre.Registre{}

	// génère notre propre requête (SiteA)
	_ = ctrl.DistFile.SCRequestFromBaseApp()

	// on appelle directement la méthode de la file avec le bon index (1) car la fonction getSiteIndexFromID n'est pas encore implémenter
	msgAck := Message{
		Type:        ACK,
		IndexSender: 1,
		IndexDest:   0,
		ClockValue:  10, //date supérieur a la requete locale
	}

	// simule l'arrivée du ACK directement dans l'algo
	isReady := ctrl.DistFile.AckFromNetwork(msgAck)

	// vérifie si l'algo nous donne le feu vert
	if !isReady {
		t.Errorf("L'algorithme aurait dû passer isReady à true. Tab[0]: %+v, Tab[1]: %+v",
			ctrl.DistFile.Tab[0], ctrl.DistFile.Tab[1])
	}

	// simule ce que le handler ferait
	if isReady {
		liberationMsg := ctrl.DistFile.SCStopFromBaseApp()

		parserMsg, err := ctrl.FileMessageToParserMessage(liberationMsg)
		if err != nil {
			t.Fatalf("Erreur conversion message: %v", err)
		}

		encoded, err := parser.Encode(parserMsg)
		if err != nil {
			t.Fatalf("Erreur encodage: %v", err)
		}

		if !strings.Contains(encoded, "ACTION:SC_LIBERATION") {
			t.Errorf("Le message de libération est incorrect: %s", encoded)
		}
	}
}
