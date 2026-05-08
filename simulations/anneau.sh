#!/bin/sh

# On récupère le chemin absolu du dossier simulations (où on veut les logs)
LOG_DIR="$(pwd)/logs"
FIFO="/tmp/backpipe"
PROJECT_DIR="$(pwd)/src/homemadeTorrent"

N=3
rm -f "$FIFO"
mkfifo "$FIFO"

# Construction de la liste des membres
IDS=""
for i in $(seq 1 $N); do
    IDS="$IDS $i"
done

echo "Logs dirigés vers : $LOG_DIR"

# On lance l'anneau
(
  cat "$FIFO" | \
  go run "$PROJECT_DIR" 1 $IDS 2> "$LOG_DIR/1.log" | \
  go run "$PROJECT_DIR" 2 $IDS 2> "$LOG_DIR/2.log" | \
  go run "$PROJECT_DIR" 3 $IDS 2> "$LOG_DIR/3.log" > "$FIFO"
)
