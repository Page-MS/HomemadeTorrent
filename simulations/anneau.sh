#!/bin/sh

# On récupère le chemin absolu du dossier simulations (où on veut les logs)
LOG_DIR="$(pwd)/logs"
FIFO="/tmp/backpipe"
PROJECT_DIR="../src/homemadeTorrent"


N=3
rm -f "$FIFO"
mkfifo "$FIFO"
mkdir -p "$LOG_DIR"

# Supprimer les données des sites des précédentes executions
TARGET="../bin"
KEEP="baseFiles"

find "$TARGET" -mindepth 1 -path "$TARGET/$KEEP" -o -path "$TARGET/$KEEP/*" -prune -o -exec rm -rf {} +

# Construction de la liste des membres
IDS=""
for i in $(seq 1 $N); do
  IDS="$IDS $i"
done

echo "Logs dirigés vers : $LOG_DIR"

# On lance l'anneau
(
  cat "$FIFO" | \
  go run -C "$PROJECT_DIR" . 1 0 $IDS 2> "$LOG_DIR/1.log" | \
  go run -C "$PROJECT_DIR" . 2 0 $IDS 2> "$LOG_DIR/2.log" | \
  go run -C "$PROJECT_DIR" . 3 0 $IDS 2> "$LOG_DIR/3.log" > "$FIFO"
)
