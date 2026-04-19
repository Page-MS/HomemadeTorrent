package main

import (
	event_loop "HomemadeTorrent/pkg/event_loop"
	registre "HomemadeTorrent/pkg/registre"
	"fmt"
)

func main() {

	fmt.Print("Hello, World!")
	registre.InitialiseRegistre()
	event_loop.Start()
}
