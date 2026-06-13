package delay

import (
	"log"
	"time"
)

type Delay struct {
	AppliationDelay_ms int `json:"applicationDelay_ms"`
	SnapshotDelay_ms   int `json:"snapshotDelay_ms"`
	NetworkDelay_ms    int `json:"networkDelay_ms"`
}

// Default values for delay, wont get on the way on normal usage
func NewDelay() Delay {
	return Delay{}
}

func Wait(duration_ms int) {
	time.Sleep(time.Duration(duration_ms) * time.Millisecond)
}
func (d *Delay) waitNamed(name string, duration_ms int) {
	log.Printf("[DELAY] %s: waiting for %d ms\n", name, duration_ms)
	time.Sleep(time.Duration(duration_ms) * time.Millisecond)
	log.Printf("[DELAY] %s: finished waiting\n", name)
}

func (d *Delay) WaitApplicationDelay() {
	d.waitNamed("ApplicationDelay", d.AppliationDelay_ms)
}

func (d *Delay) WaitSnapshotDelay() {
	d.waitNamed("SnapshotDelay", d.SnapshotDelay_ms)
}

func (d *Delay) WaitNetworkDelay() {
	d.waitNamed("NetworkDelay", d.NetworkDelay_ms)
}
