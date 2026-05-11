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
From there, you can initiate downloads and see the state of the network

## Test scenarios
In a second terminal, run one of the test scripts:
```bash
sh sc_request.sh
sh snapshot.sh
```
These scripts simulate either:
- a critical section request scenario, or
- a snapshot scenario initiated from the last node of the ring.

These scenarios are intended to test the current state of the system logic.

## Logs
Each site writes its logs into a dedicated file:

- `1.log` → logs of site 1
- `2.log` → logs of site 2
- `3.log` → logs of site 3

These logs can be used to observe message propagation and system behavior.