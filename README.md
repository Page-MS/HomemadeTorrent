# HomemadeTorrent
This projects aims to emulate a small Torrent system in Golang.

## Simplifications

We decided on a few key simplifications and differences with the real torrent system.
- We have a unified register for all available files
- The register has a synchronization system between agents
- We implement the ability to make snapshots of the current configuration of the network including ongoing transfers
- We use sha3sum and not sha1sum to check integrity

