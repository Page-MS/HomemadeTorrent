package main

import (
	event_loop "HomemadeTorrent/pkg/event_loop"
	registre "HomemadeTorrent/pkg/registre"
	"fmt"
)

func main() {

	asciiArt := `  _  _                             _       _____                    _     
 | || |___ _ __  ___ _ __  __ _ __| |___  |_   _|__ _ _ _ _ ___ _ _| |_   
 | __ / _ \ '  \/ -_) '  \/ _\ / _\ / -_)   | |/ _ \ '_| '_/ -_) ' \  _|  
 |_||_\___/_|_|_\___|_|_|_\__,_\__,_\___|   |_|\___/_| |_| \___|_||_\__|  
  ___ ___ ___ ___ ___ ___ ___ ___ ___ ___ ___ ___ ___ ___ ___ ___ ___ ___ 
 |___|___|___|___|___|___|___|___|___|___|___|___|___|___|___|___|___|___|
                                                                          
                                                                          `

	fmt.Print("\n", asciiArt, "\n\n")
	registre.CleanUpPartsDirectory()
	registreTest := registre.NewRegistre()
	registre.MakeInitialHardcodedRegister(registreTest)
	registreTest.PrintRegister()
	currentSiteID := "Page"
	registre.InitialiseRegistre(currentSiteID, registreTest)
	event_loop.Start(registreTest.GetPeerList(), currentSiteID)
}
