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
	EstampClock *clock.LamportClock
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

// Renvoie une instance de file répartie
// n le nombre de site du réseau, siteIndex l'index du site host de cette instance
func GetNewDistributedFile(n int, siteIndex int, estampClock *clock.LamportClock) *DistributedFile {
	df := &DistributedFile{
		EstampClock: estampClock,
		Tab:         make([]TabEntry, n),
		SiteIndex:   siteIndex,
	}

	for i := 0; i < n; i++ {
		df.Tab[i] = TabEntry{
			Type: SC_LIBERATION,
			Date: 0,
		}
	}

	return df
}

// Traite une demande de section critique venant de l'app du site
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

// Traite une demande de libération de section critique venant de l'app du site
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

// Traite une demande de section critique ne venant pas de l'app du site
// Renvoie le message de réponse à cette requete et un boolen indiquant si l'app du site peut entrer en section critique
// msg la requete à traiter
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

// Traite une demande de libération de section critique ne venant pas de l'app du site
// Renvoie un boolen indiquant si l'app du site peut entrer en section critique
// msg la requete à traiter
func (df *DistributedFile) SCStopFromNetwork(msg Message) bool {
	df.EstampClock.Update(msg.ClockValue)
	df.Tab[msg.IndexSender] = TabEntry{
		Type: SC_LIBERATION,
		Date: msg.ClockValue,
	}

	isScReadyForApp := false
	if df.Tab[df.SiteIndex].Type == SC_REQUEST && compareTab(df.Tab, df.SiteIndex) {
		isScReadyForApp = true
	}

	return isScReadyForApp
}

// Traite un accusé de reception ne venant pas de l'app du site.
// Renvoie un boolen indiquant si l'app du site peut entrer en section critique.
// msg la requete à traiter
func (df *DistributedFile) AckFromNetwork(msg Message) bool {
	df.EstampClock.Update(msg.ClockValue)

	if df.Tab[msg.IndexSender].Type != SC_REQUEST {
		df.Tab[msg.IndexSender] = TabEntry{
			Type: ACK,
			Date: msg.ClockValue,
		}
	}

	isScReadyForApp := false
	if df.Tab[df.SiteIndex].Type == SC_REQUEST && compareTab(df.Tab, df.SiteIndex) {
		isScReadyForApp = true
	}

	return isScReadyForApp
}

func compareTab(tab []TabEntry, siteIndex int) bool {
	reqSite := Request{tab[siteIndex].Date, siteIndex}
	for i := 0; i < len(tab); i++ {
		if i != siteIndex {
			req := Request{tab[i].Date, i}
			if !EstPrioritaire(reqSite, req) {
				return false
			}
		}
	}
	return true
}
