#!/bin/sh

if [ $# -lt 1 ]; then
    echo "Erreur: ID du site necessaire en paramètre. Ex: testTorrentAction.sh SITE1"
    exit 1
fi

printf "ACTION:TransferRelatedMessage\nDEST:$1\nSENDER:2\nPAYLOAD_LEN:36\nMessageType;TransferRelatedMessage/n\n\n" > /tmp/backpipe