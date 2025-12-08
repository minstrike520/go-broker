package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Message represents a message from a client
type Message struct {
	conn net.Conn
	text string
}

// Server handles multiple client connections
type Server struct {
	listener   net.Listener
	messages   chan Message
	closeConns chan net.Conn
}

// NewServer creates a new server instance
func NewServer(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{
		listener:   listener,
		messages:   make(chan Message, 10),
		closeConns: make(chan net.Conn, 10),
	}, nil
}

// Start starts the server with two goroutines
func (s *Server) Start() {
	fmt.Println("Server started on", s.listener.Addr())

	// Goroutine 1: Application logic (single-threaded echo server)
	go s.applicationLogic()

	// Goroutine 2: Proxy - accepts new connections and reads from all clients
	go s.proxy()

	// Block forever
	select {}
}

// applicationLogic handles the actual server logic (echoing)
func (s *Server) applicationLogic() {
	for {
		select {
		case msg := <-s.messages:
			// Echo the message back to the client
			response := msg.text + "\n"
			_, err := msg.conn.Write([]byte(response))
			if err != nil {
				fmt.Println("Error writing to client:", err)
				s.closeConns <- msg.conn
			}
		case conn := <-s.closeConns:
			conn.Close()
			fmt.Println("Connection closed:", conn.RemoteAddr())
		}
	}
}

// proxy accepts new connections and reads from all clients
// Uses non-blocking I/O with timeouts to avoid spawning goroutines
func (s *Server) proxy() {
	// Track active connections with their readers
	type connState struct {
		conn   net.Conn
		reader *bufio.Reader
	}

	connections := make(map[net.Conn]*connState)

	for {
		// Try to accept new connection (non-blocking with short timeout)
		s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Millisecond))
		conn, err := s.listener.Accept()
		if err == nil {
			fmt.Println("New connection from:", conn.RemoteAddr())
			connections[conn] = &connState{
				conn:   conn,
				reader: bufio.NewReader(conn),
			}
		}

		// Read from all connections (non-blocking with short timeout)
		for conn, state := range connections {
			conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
			line, err := state.reader.ReadString('\n')

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// No data available, continue to next connection
					continue
				}
				// Connection closed or real error
				if err != io.EOF {
					fmt.Println("Error reading from client:", err)
				}
				delete(connections, conn)
				s.closeConns <- conn
				continue
			}

			// Got a message
			text := strings.TrimSpace(line)
			if text == "" {
				continue
			}

			fmt.Printf("Received from %s: %s\n", conn.RemoteAddr(), text)

			// Check for goodbye
			if strings.ToLower(text) == "goodbye" {
				fmt.Println("Client sent goodbye:", conn.RemoteAddr())
				delete(connections, conn)
				s.closeConns <- conn
				continue
			}

			// Send message to application logic
			s.messages <- Message{
				conn: conn,
				text: text,
			}
		}
	}
}

func main() {
	server, err := NewServer(":8080")
	if err != nil {
		fmt.Println("Error creating server:", err)
		return
	}

	server.Start()
}
