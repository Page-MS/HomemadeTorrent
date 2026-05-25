package delay

import (
	"log"
	"time"
)

type Delay struct {
	AppliationDelay_ms int
	SnapshotDelay_ms int
	NetworkDelay_ms int
}

// Default values for delay, wont get on the way on normal usage
func NewDelay() Delay {
	return Delay{};
}

func Wait(duration_ms int) {
	time.Sleep(time.Duration(duration_ms) * time.Millisecond);
}

func (d *Delay) WaitApplicationDelay() {
	log.Printf("[DELAY] Wait for abritrary delay of %d ms\n", d.AppliationDelay_ms);
	time.Sleep(time.Duration(d.AppliationDelay_ms) * time.Millisecond);
	log.Printf("[DELAY] Wait for abritrary delay\n");
}

func (d *Delay) WaitSnapshotDelay() {
	log.Printf("[DELAY] Wait for abritrary delay of %d ms\n", d.AppliationDelay_ms);
	time.Sleep(time.Duration(d.SnapshotDelay_ms) * time.Millisecond);
	log.Printf("[DELAY] Wait for abritrary delay\n");
}

func (d *Delay) WaitNetworkDelay() {
	log.Printf("[DELAY] Wait for abritrary delay of %d ms\n", d.AppliationDelay_ms);
	time.Sleep(time.Duration(d.NetworkDelay_ms) * time.Millisecond);
	log.Printf("[DELAY] Wait for abritrary delay\n");
}
