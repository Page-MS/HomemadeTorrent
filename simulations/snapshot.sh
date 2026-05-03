#!/bin/sh

# Simuler une demande de snapshot émise par le site 3
printf "ACTION:MARKER\nID:snap_test\nDEST:1\nSENDER:EXTERNAL\nVECT:0,0,0\n\n" > /tmp/backpipe