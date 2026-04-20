package clock

import "sync"

type VectorClock struct {
	vector []int
	id     int // indice du site dans le tableau vector
	mu     sync.Mutex
}

// NewVectorClock initialise l'horloge avec le nombre de sites total
func NewVectorClock(nbSites int, siteID int) *VectorClock {
	return &VectorClock{
		vector: make([]int, nbSites),
		id:     siteID,
	}
}

// Tick incrémente la case correspondant à ce site
func (vc *VectorClock) Tick() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.vector[vc.id]++
}

// Update prend le max de chaque case entre le vecteur local et reçu (regle de Lamport du cours)
func (vc *VectorClock) Update(remoteVector []int) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Verification de la taille des vecteurs
	for i := range vc.vector {
		if remoteVector[i] > vc.vector[i] {
			vc.vector[i] = remoteVector[i]
		}
	}
	// On incrémente aussi notre propre case
	vc.vector[vc.id]++
}

// GetCopy retourne une copie du vecteur actuel (pour l'envoyer dans un message)
func (vc *VectorClock) GetCopy() []int {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	copyVec := make([]int, len(vc.vector))
	copy(copyVec, vc.vector)
	return copyVec
}
