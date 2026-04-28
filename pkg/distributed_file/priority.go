package distributed_file

// Request est une structure simplifiée (permet au controleur de comparer des requetes sans avoir a manipuler le message complet)
type Request struct {
	Stamp     int // estampille de Lamport extraite du message
	SiteIndex int
}

// EstPrioritaire pour avoir l'ordre total
// renvoie true si reqA passe avant reqB.
func EstPrioritaire(reqA, reqB Request) bool {
	if reqA.Stamp < reqB.Stamp {
		return true
	}
	if reqA.Stamp > reqB.Stamp {
		return false
	}
	return reqA.SiteIndex < reqB.SiteIndex
}
