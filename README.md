# HomemadeTorrent
This projects aims to emulate a small Torrent system in Golang.

## Simplifications

We decided on a few key simplifications and differences with the real torrent system.
- We have a unified shared register for all available files
- The register has a synchronization system between agents
- We implement the ability to make snapshots of the current configuration of the network including ongoing transfers
- We use sha3sum and not sha1sum to check integrity
- The current network structure is a ring

# How to use
This guide assumes a Linux terminal with Go installed.  
Once the project is imported, move into the `simulations/` directory:

```bash
cd simulations/
```

## Ring
In a first terminal, run the `anneau.sh` script, which starts a ring of 3 sites:
```bash
sh anneau.sh
```
## UI
You can then open in a browser the link that appeared in the logs.
From there, you can initiate downloads and see the state of the network. The adresses are usually :
- Site 1: `http://localhost:8080`
- Site 2: `http://localhost:8081`
- Site 3: `http://localhost:8082`

## Test scenarios
Once in the ui, you can download the files not present on your site.

## Logs
Each site writes its logs into a dedicated file:

- `1.log` → logs of site 1
- `2.log` → logs of site 2
- `3.log` → logs of site 3

These logs can be used to observe message propagation and system behavior. You can find them in the simulation folder.

# Project Structure

**`src/`** — Application entry point (`main.go`) that initializes each site with its configuration

**`pkg/`** — Core packages:
- `registre/` — Centralized metadata registry tracking available files and their distribution
- `event_loop/` — Main dispatcher routing events between UI, network, and torrent logic
- `control/` — Network message handling, synchronization, and state snapshots
- `torrentLogic/` — File transfer coordination and shasum verification
- `distributed_file/` — File part prioritization logic
- `webui/` — Web interface for file downloads and network monitoring
- `clock/` — Lamport and Vector clocks for distributed event ordering
- `snapshot/` — Network state capture for debugging and recovery

**`bin/`** — Runtime data: base files, file parts (16KB chunks), and site-specific transfer directories

**`simulations/`** — Test scenarios, including `anneau.sh` that launches the 3-site ring network with logging

## How It Works

**Network Topology:** The system operates as a ring of 3 autonomous sites, each running independently but connected through a distributed registry and message-passing protocol.

**Initialization:** When the simulation starts (`anneau.sh`), each site:
- Loads file metadata from a centralized registry
- Initializes pre-configured files in its local directory (`bin/1/`, `bin/2/`, `bin/3/`)
- Starts the event loop that handles UI requests, network messages, and file transfers

**File Transfer Protocol:** When a user requests a file not available locally:
1. The site requests a shasum (checksum) of the desired file parts from peers
2. Upon verification, it requests the actual file content from the peer holding it
3. File integrity is verified using SHA256 checksums
4. Downloaded parts are cached locally for future sharing with other sites

**Distributed Coordination:**
- **Registry:** Maintains a unified view of all files and which sites hold which parts
- **Clocks:** Lamport and Vector clocks ensure causality ordering across the ring
- **Snapshots:** Network state can be captured at any time for debugging and recovery

**Event Loop:** The central event loop orchestrates four concurrent listeners:
- Input from others sites on std entry
- Requests from the web UI
- Messages from the torrent logic (transfer completions, etc.)

This architecture ensures coordinated file distribution while allowing each site to operate independently and maintaining consistency across the distributed system.


