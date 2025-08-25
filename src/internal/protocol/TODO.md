Rethink what role this package serves. Implementing networking in common code is
probably a good idea, as generally we want either server sending replicas to
clients, or sending a serialized stream of input commands to all clients, but
it'd also be nice to not close the gate for whatever alternative approaches.
