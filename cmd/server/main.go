package main

import (
    "bufio"
    "fmt"
    "net"
    "strings"
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
    newConns   chan net.Conn
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
        newConns:   make(chan net.Conn, 10),
        closeConns: make(chan net.Conn, 10),
    }, nil
}

// Start starts the server with two goroutines
func (s *Server) Start() {
    fmt.Println("Server started on", s.listener.Addr())

    // Goroutine 1: Application logic (single-threaded echo server)
    go s.applicationLogic()

    // Goroutine 2: Proxy - accepts new connections and messages
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

// proxy accepts new connections and routes messages
func (s *Server) proxy() {
    // Accept new connections
    go func() {
        for {
            conn, err := s.listener.Accept()
            if err != nil {
                fmt.Println("Error accepting connection:", err)
                continue
            }
            s.newConns <- conn
        }
    }()

    // Handle new connections
    for conn := range s.newConns {
        fmt.Println("New connection from:", conn.RemoteAddr())
        go s.handleConnection(conn)
    }
}

// handleConnection reads messages from a client connection
func (s *Server) handleConnection(conn net.Conn) {
    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        text := strings.TrimSpace(scanner.Text())

        if text == "" {
            continue
        }

        fmt.Printf("Received from %s: %s\n", conn.RemoteAddr(), text)

        // Check for goodbye
        if strings.ToLower(text) == "goodbye" {
            fmt.Println("Client sent goodbye:", conn.RemoteAddr())
            s.closeConns <- conn
            return
        }

        // Send message to application logic
        s.messages <- Message{
            conn: conn,
            text: text,
        }
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading from client:", err)
    }

    s.closeConns <- conn
}

func main() {
    server, err := NewServer(":8080")
    if err != nil {
        fmt.Println("Error creating server:", err)
        return
    }

    server.Start()
}
