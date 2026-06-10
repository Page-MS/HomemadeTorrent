#!/bin/sh

LOG_DIR="$(pwd)/logs"
PROJECT_DIR="../src/homemadeTorrent"
BIN_DIR="../bin"
FIFO_DIR="/tmp/network_fifos"

# Nettoyage et setup des dossiers
mkdir -p "$LOG_DIR"
mkdir -p "$FIFO_DIR"
rm -f "$FIFO_DIR"/in* "$FIFO_DIR"/out*
find "$BIN_DIR" -mindepth 1 -not -path "$BIN_DIR/baseFiles*" -exec rm -rf {} + 2>/dev/null

# Liste des noeuds et du nombre de voisins
node="node8"

# Création des named pipes
mkfifo "$FIFO_DIR/in_$node"
mkfifo "$FIFO_DIR/out_$node"


# ============ LANCEMENT ===================

echo "Logs dirigés vers : $LOG_DIR"

# Lancement du noeuds
go run -C "$PROJECT_DIR" . "$node" "0" 1 $node \
    < "$FIFO_DIR/in_$node" \
    > "$FIFO_DIR/out_$node" \
    2> "$LOG_DIR/$node.log" &


# Délai pour laisser le noeuds démarrer
sleep 1

# Topologie
# 8 -> 8
cat "$FIFO_DIR/out_node8" > "$FIFO_DIR/in_node8" &

echo "Réseau démarré. Ctrl+C pour arrêter."

# Attente et nettoyage à l'arrêt
cleanup() {
    echo "Arrêt du réseau..."
    kill $(jobs -p) 2>/dev/null
    wait 2>/dev/null
    echo "Nodes arrêtés."
}
trap cleanup EXIT INT TERM
wait