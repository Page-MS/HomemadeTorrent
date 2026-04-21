package control

import "testing"

func TestEstPrioritaire(t *testing.T) {
	// Estampilles différentes
	reqA := Request{Stamp: 5, SiteID: "SiteB"}
	reqB := Request{Stamp: 10, SiteID: "SiteA"}

	if !EstPrioritaire(reqA, reqB) {
		t.Errorf("Erreur Cas 1 : La plus petite estampille (5) devrait être prioritaire sur (10)")
	}

	// Estampilles identiques -> arbitrage par ID
	reqC := Request{Stamp: 10, SiteID: "SiteA"}
	reqD := Request{Stamp: 10, SiteID: "SiteB"}

	if !EstPrioritaire(reqC, reqD) {
		t.Errorf("Erreur Cas 2 : À estampilles égales, SiteA devrait gagner contre Site_B")
	}

	if EstPrioritaire(reqD, reqC) {
		t.Errorf("Erreur Cas 2 : SiteB ne devrait pas être prioritaire sur SiteA à estampille égale")
	}

	// Même ID, estampilles différentes (au cas ou)
	reqE := Request{Stamp: 8, SiteID: "SiteA"}
	reqF := Request{Stamp: 9, SiteID: "SiteA"}

	if !EstPrioritaire(reqE, reqF) {
		t.Errorf("Erreur Cas 3 : La plus petite estampille devrait gagner à ID égal")
	}
}
