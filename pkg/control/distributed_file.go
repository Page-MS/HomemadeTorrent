package control

import (
	"HomemadeTorrent/pkg/clock"
	"strconv"
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
	SiteIndex   int // Conversion Id site en index dans la logique du controleur
}

// ------------- Structure Message traité par la file ----------------------
// La conversion du message brut reçut en message de ce type ce fait dans la logique du controleur en amont
type Message struct {
	Type        MessageType
	IndexSender int // Conversion Id site en index dans la logique du controleur
	IndexDest   int
	ClockValue  int
}

func GetNewDistributedFile(n int, SiteIndex int) *DistributedFile {
	df := &DistributedFile{
		EstampClock: clock.LamportClock{},
		Tab:         make([]TabEntry, n),
		SiteIndex:   SiteIndex,
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
	df.Tab[df.SiteIndex] = TabEntry{
		Type: SC_REQUEST,
		Date: df.EstampClock.GetValue(),
	}

	return Message{
		Type:        SC_REQUEST,
		IndexSender: df.SiteIndex,
		IndexDest:   -1, // Index du broadcast
		ClockValue:  df.EstampClock.GetValue(),
	}
}

func (df *DistributedFile) SCStopFromBaseApp() Message {
	df.EstampClock.Tick()
	df.Tab[df.SiteIndex] = TabEntry{
		Type: SC_LIBERATION,
		Date: df.EstampClock.GetValue(),
	}

	return Message{
		Type:        SC_LIBERATION,
		IndexSender: df.SiteIndex,
		IndexDest:   -1, // Index du broadcast
		ClockValue:  df.EstampClock.GetValue(),
	}
}

func (df *DistributedFile) SCRequestFromNetwork(msg Message) (Message, bool) {
	df.EstampClock.Update(msg.ClockValue)
	df.Tab[msg.IndexSender] = TabEntry{
		Type: SC_REQUEST,
		Date: msg.ClockValue,
	}

	ack := Message{
		Type:        ACK,
		IndexSender: df.SiteIndex,
		IndexDest:   msg.IndexSender,
		ClockValue:  df.EstampClock.GetValue(),
	}

	isScReadyForApp := false
	if df.Tab[df.SiteIndex].Type == SC_REQUEST && compareTab(df.Tab, df.SiteIndex) {
		isScReadyForApp = true
	}

	return ack, isScReadyForApp
}

func compareTab(tab []TabEntry, siteIndex int) bool {
	reqSite := Request{tab[siteIndex].Date, strconv.Itoa(siteIndex)}
	for i := 0; i < len(tab); i++ {
		if i != siteIndex {
			req := Request{tab[i].Date, strconv.Itoa(i)}
			if !EstPrioritaire(reqSite, req) {
				return false
			}
		}
	}
	return true
}
