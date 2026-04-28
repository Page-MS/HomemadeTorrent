package distributed_file

import "testing"

func TestEstPrioritaire(t *testing.T) {
	// Estampilles différentes
	reqA := Request{Stamp: 5, SiteIndex: 2}
	reqB := Request{Stamp: 10, SiteIndex: 1}

	if !EstPrioritaire(reqA, reqB) {
		t.Errorf("Erreur Cas 1 : La plus petite estampille (5) devrait être prioritaire sur (10)")
	}

	// Estampilles identiques -> arbitrage par ID
	reqC := Request{Stamp: 10, SiteIndex: 1}
	reqD := Request{Stamp: 10, SiteIndex: 2}

	if !EstPrioritaire(reqC, reqD) {
		t.Errorf("Erreur Cas 2 : À estampilles égales, SiteA devrait gagner contre Site_B")
	}

	if EstPrioritaire(reqD, reqC) {
		t.Errorf("Erreur Cas 2 : SiteB ne devrait pas être prioritaire sur SiteA à estampille égale")
	}

	// Même ID, estampilles différentes (au cas ou)
	reqE := Request{Stamp: 8, SiteIndex: 1}
	reqF := Request{Stamp: 9, SiteIndex: 1}

	if !EstPrioritaire(reqE, reqF) {
		t.Errorf("Erreur Cas 3 : La plus petite estampille devrait gagner à ID égal")
	}
}
