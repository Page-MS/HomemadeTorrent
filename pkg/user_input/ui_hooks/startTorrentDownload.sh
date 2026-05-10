#!/bin/sh

if [ $# -lt 2 ]; then
    echo "Erreur: ID du site et du fichier necessaires en paramètre. Ex: startSnapshot.sh SITE1 FILE1"
    echo "Usage: SiteID FileID"
    exit 1
fi

SITEID="$1"
FILEID="$2"

# Construction du payload
PAYLOAD="FileID;$FILEID/n"

# Calcul de la longueur
PAYLOAD_LEN=${#PAYLOAD}


printf "ACTION:StartTransfers\nDEST:$SITEID\nPAYLOAD_LEN:$PAYLOAD_LEN\n$PAYLOAD\n\n" > /tmp/site_input_$1