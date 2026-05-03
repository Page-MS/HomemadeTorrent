#!/bin/sh

N=3
FIFO=/tmp/backpipe

# Nettoyage ancien FIFO
[ -p "$FIFO" ] && rm "$FIFO"

mkfifo "$FIFO"

# Construction des IDs
IDS=""
i=1
while [ $i -le $N ]; do
    IDS="$IDS $i"
    i=$((i+1))
done

echo "Nodes:$IDS"

# Construction du pipeline
CMD=""

i=1
while [ $i -le $N ]; do
    if [ $i -eq 1 ]; then
        CMD="go run ../src/homemadeTorrent/main.go $i$IDS < $FIFO"
    elif [ $i -eq $N ]; then
        CMD="$CMD | go run ../src/homemadeTorrent/main.go $i$IDS > $FIFO"
    else
        CMD="$CMD | go run ../src/homemadeTorrent/main.go $i$IDS"
    fi
    i=$((i+1))
done

echo "Running:"
echo "$CMD"
eval "$CMD"