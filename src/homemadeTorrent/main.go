package main

import (
	event_loop "HomemadeTorrent/pkg/event_loop"
)

func main() {

	allSiteIDs := []string{"Test"}
	event_loop.Start(allSiteIDs, "Test")
}
