# Registre Module

This module hold the logic for the register at the center of the torrent simulation.

This register is shared and synchronized between all sites.

## Initialisation 

Because this is only a simulation. All files are available in the /bin repository. The register starts with a base state that will then change over time. When the site loads, depending of it's id, it's gonna check the register and directly copy all the files that they are supposed to hold inside a subfolder of /bin which is named accordingly to their id.

## Content of the register

The register holds the file list, how a file should be broken down and the checksum of all file parts; the hosts that hold a copy of them.

The piece of a file are 16kb long except for the last one which can be shorter


