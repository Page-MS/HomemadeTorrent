#!/bin/sh

# Pour tests seulement car pas de scénarios avec une demande SC de l'utilisateur

if [ $# -lt 1 ]; then
    echo "Erreur: ID du site necessaire en paramètre. Ex: startSC.sh SITE1"
    exit 1
fi

printf "ACTION:LOCAL_SC_REQUEST\nDEST:$1\n\n" > /tmp/site_input_$1