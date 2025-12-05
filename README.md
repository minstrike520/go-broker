# Go Broker Implementation

A simple broker implementation following the specifications in `request.md`.

## Features

- Server with exactly two goroutines:
  - **Proxy goroutine**: Accepts new connections and routes messages
  - **Application logic goroutine**: Single-threaded echo server
- Echo server: Returns the same message the client sends
- Special command: "goodbye" terminates the connection
- Supports multiple concurrent clients

## Architecture

The implementation uses a channel-based message passing system:
- `messages` channel: Routes client messages to the application logic
- `newConns` channel: Handles new client connections
- `closeConns` channel: Manages connection cleanup

## Running

### Start the server:
```bash
go run cmd/server/main.go
```

### Start a client (in a separate terminal):
```bash
go run cmd/client/main.go
```

### Test with multiple clients:
Open multiple terminals and run `go run cmd/client/main.go` in each.

### Run automated test:
```bash
./test.sh
```
(Note: Make sure the server is running first)

## Usage Example

```
Client: hello
Server: hello

Client: world
Server: world

Client: goodbye
Connection closed
```

## Implementation Notes

- The server is single-threaded for application logic (as per specs)
- Connection handling is delegated to separate goroutines spawned by the proxy
- Clean separation between networking (proxy) and application logic (echo)
