package clock

import (
	"sync"
	"testing"
)

func TestLamportClock(t *testing.T) {
	clock := &LamportClock{}

	// Test de l'état initial
	if clock.GetValue() != 0 {
		t.Errorf("L'horloge devrait démarrer à 0, reçu: %d", clock.GetValue())
	}

	// Test du Tick (événement local ou envoi)
	val := clock.Tick()
	if val != 1 {
		t.Errorf("Après un Tick, l'horloge devrait être à 1, reçu: %d", val)
	}

	// Test de l'Update
	// Si on reçoit une estampille 10 alors qu'on est à 1
	clock.Update(10)
	if clock.GetValue() != 11 {
		t.Errorf("Après Update(10), l'horloge devrait être à 11 (max(1, 10)+1), reçu: %d", clock.GetValue())
	}

	// Test de l'Update
	// Si on reçoit une estampille 5 alors qu'on est déjà à 11
	clock.Update(5)
	if clock.GetValue() != 12 {
		t.Errorf("Après Update(5) étant à 11, l'horloge devrait être à 12, reçu: %d", clock.GetValue())
	}
}

// 5. Test de concurrence (pour vérifier si le mutx fonctionne)
func TestClockConcurrency(t *testing.T) {
	clock := &LamportClock{}
	var wg sync.WaitGroup
	iterations := 1000

	wg.Add(2)

	// Deux goroutines qui incrémentent l'horloge massivement en même temps
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			clock.Tick()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			clock.Update(i)
		}
	}()

	wg.Wait()

	// Grace au Mutex on garanti l'intégrité de l'horloge
	if clock.GetValue() <= iterations {
		t.Errorf("La valeur finale semble trop basse, possible problème de concurrence")
	}
}
