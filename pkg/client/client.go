package client

import (
	bootstrapingnode "HomemadeTorrent/pkg/bootstraping_node"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"slices"
	"strings"
)

// connects to a known peer and return a list of clients
//
// The boostraping node returns clients in a string with format:
//
// 192.168.1.89,192.168...,...
func bootstrap() []string {
	c, err := net.Dial("tcp", bootstrapingnode.BootstrapNodeAddr)
	if err != nil {
		log.Fatal("Unabled to open tcp connexion: ", err)
	}
	defer c.Close()

	res := make([]string, 10)
	buf := make([]byte, 1024)
	for {
		nb, err := c.Read(buf)
		if err == io.EOF { // server close the connexion
			break
		} else if err != nil {
			log.Fatal("Error while reading response: ", err)
		}
		res = append(res, string(buf[:nb]))
	}

	clients := strings.Split(strings.Join(res, ""), ",")
	for index  := range clients {
		trimed := strings.TrimSpace(clients[index])
		clients[index] = trimed
	}

	// definitely overkill but osef
	ipRegex := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	clients = slices.DeleteFunc(clients, func(s string) bool {
		return len(s) <= 0 || !ipRegex.MatchString(s)
	})
	fmt.Println("Received clients:", clients)
	return clients
}

