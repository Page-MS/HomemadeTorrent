package control

import (
	"HomemadeTorrent/pkg/clock"
)

// ------------- Types et Structures file -----------------------
type MessageType string

const (
	SC_REQUEST    MessageType = "SC_REQUEST"
	SC_LIBERATION MessageType = "SC_LIBERATION"
	ACK           MessageType = "ACK"
)

type TabEntry struct {
	Type MessageType
	Date int
}

type DistributedFile struct {
	EstampClock clock.LamportClock
	Tab         []TabEntry
	siteIndex   int
}

// ------------- Structure Message traité par la file ----------------------
// La conversion du message brut reçut en message de ce type ce fait dans la logique du controleur en amont
type Message struct {
	Type        MessageType
	indexSender int
}

func GetNewDistributedFile(n int, siteIndex int) *DistributedFile {
	df := &DistributedFile{
		EstampClock: clock.LamportClock{},
		Tab:         make([]TabEntry, n),
		siteIndex:   siteIndex,
	}

	for i := 0; i < n; i++ {
		df.Tab[i] = TabEntry{
			Type: SC_LIBERATION,
			Date: 0,
		}
	}

	return df
}

func (df *DistributedFile) SCRequestFromBaseApp() Message {
	df.EstampClock.Tick()
	df.Tab[df.siteIndex] = TabEntry{
		Type: SC_REQUEST,
		Date: df.EstampClock.GetValue(),
	}

	return Message{
		Type:        SC_REQUEST,
		indexSender: df.siteIndex,
	}
}
