#!/bin/sh

if [ $# -lt 1 ]; then
    echo "Erreur: ID du site necessaire en paramètre. Ex: startSnapshot.sh SITE1"
    exit 1
fi

printf "ACTION:MARKER\nID:snap_start\nDEST:$1\nSENDER:USER\n\n" > /tmp/site_input_$1