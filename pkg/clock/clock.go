package clock

import "sync"

type LamportClock struct {
	counter int
	mu      sync.Mutex
}

// Tick incrémente l'horloge avant un envoi ou pour un événement local
func (lc *LamportClock) Tick() int {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.counter++
	return lc.counter
}

// Update applique la règle de Lamport (cours) : max(local, distant) + 1
func (lc *LamportClock) Update(remoteStamp int) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if remoteStamp > lc.counter {
		lc.counter = remoteStamp
	}
	lc.counter++
}

// GetValue permet de lire l'heure logique actuelle en toute sécurité
func (lc *LamportClock) GetValue() int {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.counter
}
