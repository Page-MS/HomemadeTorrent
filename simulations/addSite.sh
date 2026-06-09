#!/bin/sh

LOG_DIR="$(pwd)/logs"
PROJECT_DIR="../src/homemadeTorrent"
FIFO_DIR="/tmp/network_fifos"

# Setup des dossiers
mkdir -p "$LOG_DIR"
mkdir -p "$FIFO_DIR"

node="node8"

rm -f "$FIFO_DIR"/in_$node "$FIFO_DIR"/out_$node
mkfifo "$FIFO_DIR/in_$node"
mkfifo "$FIFO_DIR/out_$node"

# Lancement du noeud bootstrap, il ne connait pas la liste des autres nœuds
go run -C "$PROJECT_DIR" . "$node" 0 1 ""\
    < "$FIFO_DIR/in_$node" \
    > "$FIFO_DIR/out_$node" \
    2> "$LOG_DIR/$node.log" &

GO_PID=$!
# Attendre que le binaire enfant soit lancé par go run
sleep 1
BIN_PID=$(pgrep -P $GO_PID)

echo "Site lancé. Ctrl+C pour arrêter."

# Attente et nettoyage à l'arrêt
cleanup() {
    echo "Arrêt du site..."
    kill $BIN_PID 2>/dev/null
    kill $GO_PID 2>/dev/null
    wait 2>/dev/null
    echo "Node arrêté."
}
trap cleanup EXIT INT TERM
wait