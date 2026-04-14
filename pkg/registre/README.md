# Registre Module

This module hold the logic for the register at the center of the torrent simulation.

This register is shared and synchronized between all sites.

## Initialisation 

Because this is only a simulation. All files are available in the /bin repository. When the site loads, depending of it's id, it's gonna check the register and directly copy all the files that they are supposed to hold inside a subfolder of /bin which is named accordingly to their id.

