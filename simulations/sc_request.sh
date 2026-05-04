#!/bin/sh

# Simuler un message émis par le troisième site demande une section critique
printf "ACTION:SC_REQUEST\nID:msg-1\nDEST:-1\nSENDER:3\nSTAMP:1\nVECT:0,0,1\n\n" > /tmp/backpipe