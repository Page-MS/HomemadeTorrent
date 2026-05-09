# BUG reglés
- Enlever le clean car sinon ils se construisent et se clean en meme temps et entre eux (à faire séparément)
- TransfersID ajouté à HandlePeerAskingIfWeHavePart et HandlePeerAskingForPartContent car sinon on dupliquait les transfers
- Chemins des fichiers dans registre (../../bin au lieu de ./bin)

# BUG non réglés
- Il essaye de check à chaque fois une part 0 qui n'existe jamais
- Deadlock des go-routines -> le transfert ne se finit jamais -> on passe jamais le wg.wait() dans StartOutgoingTransfer