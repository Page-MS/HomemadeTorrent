package clock

import (
	"log"
	"sync"
)

type VectorClock struct {
	vector    []int
	siteIndex int // indice du site dans le tableau vector
	mu        sync.Mutex
}

// NewVectorClock initialise l'horloge avec le nombre de sites total
func NewVectorClock(nbSites int, siteIndex int) *VectorClock {
	return &VectorClock{
		vector:    make([]int, nbSites),
		siteIndex: siteIndex,
	}
}

// Tick incrémente la case correspondant à ce site
func (vc *VectorClock) Tick() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.vector[vc.siteIndex]++
}

// Update prend le max de chaque case entre le vecteur local et reçu (regle de Lamport du cours)
func (vc *VectorClock) Update(remoteVector []int) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	log.Printf("[DEBUG-VECT] RECEPTION - Avant Update: %v | Remote reçu: %v | MonIndex: %d", vc.vector, remoteVector, vc.siteIndex)

	if len(remoteVector) != len(vc.vector) {
		log.Printf("[DEBUG-VECT] WARNING : Taille incohérente ! Local=%d, Remote=%d. Résolution automatique...", len(vc.vector), len(remoteVector))

		if len(remoteVector) < len(vc.vector) {
			// Le message vient de l'ancienne topologie : on crée un vecteur à la nouvelle taille rempli de 0
			extended := make([]int, len(vc.vector))
			copy(extended, remoteVector) // On copie les anciennes valeurs dedans
			remoteVector = extended
		} else {
			// Si jamais le remote est plus grand, on le tronque à notre taille locale
			remoteVector = remoteVector[:len(vc.vector)]
		}
	}

	// Verification de la taille des vecteurs
	for i := range vc.vector {
		if remoteVector[i] > vc.vector[i] {
			vc.vector[i] = remoteVector[i]
		}
	}
	// On incrémente aussi notre propre case
	vc.vector[vc.siteIndex]++

	log.Printf("[DEBUG-VECT] RECEPTION - Après Update: %v", vc.vector)
}

// GetCopy retourne une copie du vecteur actuel (pour l'envoyer dans un message)
func (vc *VectorClock) GetCopy() []int {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	copyVec := make([]int, len(vc.vector))
	copy(copyVec, vc.vector)
	return copyVec
}

// UpdateLayout remplace le vecteur et l'index (indispensable lors d'une restructuration du réseau)
func (vc *VectorClock) UpdateLayout(newVector []int, newSiteIndex int) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.vector = newVector
	vc.siteIndex = newSiteIndex
}
