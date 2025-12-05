# Broker Implementation

## Relating to Networking

The project consists of "server" and "client" parts.

One server serves for multiple clients.

## Application Logic

Echoing: Server always reply with the same word that a client sends, except "goodbye" which terminates a connection.

## Implementation Specs

- Do not make things unnecessarily complicate.

- Use only TWO goroutines,
    - one serving as a proxy in that it accepts new connections as well as messages from existing connections, 
    - the other one performing the actual server application logic.

- Server is single-threaded; i.e. it's of *reactor pattern*.
