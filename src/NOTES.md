# Decoupling games from the common code and rendering

Networking should be handled by the client and server. Rendering glue needs to
be aware of the game it's working with, so the game should export things that
are necessary for that.
