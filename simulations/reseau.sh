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
NODES="node1 node2 node3 node4 node5 node6 node7"
IDS="$NODES"
DEGREES="2 3 2 3 2 2 2"

# Création des named pipes
for node in $NODES; do
    mkfifo "$FIFO_DIR/in_$node"
    mkfifo "$FIFO_DIR/out_$node"
done

# ============ LANCEMENT ===================

echo "Logs dirigés vers : $LOG_DIR"

# Lancement des noeuds
i=1
for node in $NODES; do
    degree=$(echo "$DEGREES" | cut -d' ' -f$i)
    go run -C "$PROJECT_DIR" . "$node" "$degree" 0 $IDS \
        < "$FIFO_DIR/in_$node" \
        > "$FIFO_DIR/out_$node" \
        2> "$LOG_DIR/$node.log" &
    i=$((i + 1))
done

# Délai pour laisser les noeuds démarrer
sleep 1

# Topologie
# 1 -> 2, 3
cat "$FIFO_DIR/out_node1" | tee "$FIFO_DIR/in_node2" "$FIFO_DIR/in_node3" > /dev/null &
# 2 -> 1, 4, 5
cat "$FIFO_DIR/out_node2" | tee "$FIFO_DIR/in_node1" "$FIFO_DIR/in_node4" "$FIFO_DIR/in_node5" > /dev/null &
# 3 -> 1, 6
cat "$FIFO_DIR/out_node3" | tee "$FIFO_DIR/in_node1" "$FIFO_DIR/in_node6" > /dev/null &
# 4 -> 2, 5, 7
cat "$FIFO_DIR/out_node4" | tee "$FIFO_DIR/in_node2" "$FIFO_DIR/in_node5" "$FIFO_DIR/in_node7" > /dev/null &
# 5 -> 2, 4
cat "$FIFO_DIR/out_node5" | tee "$FIFO_DIR/in_node2" "$FIFO_DIR/in_node4" > /dev/null &
# 6 -> 3, 7
cat "$FIFO_DIR/out_node6" | tee "$FIFO_DIR/in_node3" "$FIFO_DIR/in_node7" > /dev/null &
# 7 -> 4, 6
cat "$FIFO_DIR/out_node7" | tee "$FIFO_DIR/in_node4" "$FIFO_DIR/in_node6" > /dev/null &

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