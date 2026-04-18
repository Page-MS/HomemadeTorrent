package bootstrapingnode

import (
	"net"
	"fmt"
)

const BootstrapNodeAddr = "localhost:6767"

// TODO: remove this hardcoded list
const connectedClients = "192.168.1.91,     192.168.1.93,    bijour, aled "

// send known clients and close the connexion
func sendClients(conn net.Conn) {
	nb, err := conn.Write([]byte(connectedClients))
	if err != nil {
		return
	}
	fmt.Println("new conn, writed:", nb)
	conn.Close()
}

func Start() {
	port := ":6767"
	sock, err := net.Listen("tcp", port)
	if err != nil {
		return
	}
	fmt.Println("Bootstraping node running on port", port)
	for {
		c, err := sock.Accept()
		if err != nil {
			return
		}
		go sendClients(c)
	}
}

