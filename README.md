# HomemadeTorrent
This projects aims to emulate a small Torrent system in Golang.

## Simplifications

We decided on a few key simplifications and differences with the real torrent system.
- We have a unified shared register for all available files
- The register has a synchronization system between agents
- We implement the ability to make snapshots of the current configuration of the network including ongoing transfers
- We use sha3sum and not sha1sum to check integrity
- The network is dynamic: sites can join or leave while the shared register stays synchronized

# How to use
This guide assumes a Linux terminal with Go installed.  
Once the project is imported, move into the `simulations/` directory:

```bash
cd simulations/
```

## Lunch initial network
In a first terminal, run the `reseau.sh` script, which starts a network of 7 sites:
```bash
sh reseau.sh
```
## UI
You can then open in a browser the link that appeared in the logs.
From there, you can initiate downloads and see the state of the network. The adresses are usually :
- Site 1: `http://localhost:8080`
- Site 2: `http://localhost:8081`
- Site 3: `http://localhost:8082`
- ...

## Logs
Each site writes its logs into a dedicated file:

- `1.log` → logs of site 1
- `2.log` → logs of site 2
- `3.log` → logs of site 3

These logs can be used to observe message propagation and system behavior. You can find them in the simulation folder.

## Join the Network with a New Site
To add a new site to the running network, open a new terminal and run the `addSite.sh` script with the desired site name as argument:

```bash
sh addSite.sh node8
```

Once the site is running, check its log file to find its local UI address (usually `http://localhost:xxxx`), then open it in a browser.

From the UI, enter the name of an existing node (e.g. `node1`) to connect to and click the **[ START Join Network ]** button. After a short moment, click **Refresh Register** — the register should now be synchronized with the rest of the network.

## Leave the Network
To remove a site from the network, open its UI at the address shown at the top of its log file (usually `http://localhost:xxxx`), then click the **[ Leave Network ]** button.

The site's log will confirm the shutdown. If that site was exposing files, they will automatically disappear from the register on all remaining nodes — you can verify this by checking the register on any other site (e.g. after `node1` leaves, its files should no longer appear).

# Project Structure

**`src/`** — Application entry point (`main.go`) that starts each site.

**`pkg/`** — Main Go modules:
- `control/` — message handling, synchronization, snapshots
- `network_controler/` — joins, leaves, elections, neighbour discovery
- `registre/` — shared file registry and file availability tracking
- `torrentLogic/` — transfers, checksums, and file delivery
- `distributed_file/` — chunk priority and download ordering
- `webui/` — browser UI for downloads and monitoring
- `event_loop/` — central event dispatcher
- `clock/` — Lamport and vector clocks
- `snapshot/` — snapshot helpers and tests
- `delay/`, `parser/`, `utils/` — support utilities

**`bin/`** — Runtime data, sample nodes (`node1`, `node2`, `node8`) and chunk files in `parts/`.

**`simulations/`** — Launch scripts: `reseau.sh` for the main network and `addSite.sh` to join a new site.

**`snapshots/`** — Saved network snapshots produced during runs.

**`attente/`** — Temporary/queued files waiting to be processed.

## How It Works

**Network Topology:** The system runs as a dynamic set of autonomous sites. They communicate through a shared registry and event loop instead of relying on a fixed ring.

**Initialization:** When the simulation starts (`reseau.sh`), each site:
- loads its local configuration and file metadata
- starts its own event loop for UI, network, and transfer events
- becomes available to share or request files

**Joining the network:** A new site is launched with `addSite.sh`, connects to an existing node from its UI, and refreshes the register so the new peer appears in the shared view.

**Leaving the network:** A site can stop itself from the UI. The remaining nodes then remove its files from the register and continue with the current network state.

**File Transfer Protocol:** When a user requests a file not available locally:
1. The site requests a shasum (checksum) of the desired file parts from peers
2. Upon verification, it requests the actual file content from the peer holding it
3. File integrity is verified using SHA256 checksums
4. Downloaded parts are cached locally for future sharing with other sites

**Distributed Coordination:**
- **Registry:** keeps a unified view of available files and their holders
- **Clocks:** Lamport and vector clocks help order events
- **Snapshots:** capture the current state for debugging or recovery

This design keeps file sharing coordinated while allowing sites to appear and disappear dynamically.


