package main

import (
    "bufio"
    "fmt"
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

// proxy accepts new connections and routes messages from all clients
func (s *Server) proxy() {
    clients := make(map[net.Conn]*bufio.Reader)

    for {
        // Set a short deadline for Accept to allow checking for messages
        s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(100 * time.Millisecond))

        // Try to accept new connection
        conn, err := s.listener.Accept()
        if err == nil {
            fmt.Println("New connection from:", conn.RemoteAddr())
            clients[conn] = bufio.NewReader(conn)
        }

        // Read from all existing clients
        for conn, reader := range clients {
            // Set read deadline to avoid blocking
            conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))

            line, err := reader.ReadString('\n')

            if err != nil {
                // Check if it's a timeout error
                if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
                    // Timeout is expected, continue
                    continue
                }
                // Real error or EOF, close connection
                if err.Error() != "EOF" {
                    fmt.Println("Error reading from client:", err)
                }
                s.closeConns <- conn
                delete(clients, conn)
                continue
            }

            text := strings.TrimSpace(line)

            if text == "" {
                continue
            }

            fmt.Printf("Received from %s: %s\n", conn.RemoteAddr(), text)

            // Check for goodbye
            if strings.ToLower(text) == "goodbye" {
                fmt.Println("Client sent goodbye:", conn.RemoteAddr())
                s.closeConns <- conn
                delete(clients, conn)
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
    server, err := NewServer(":8000")
    if err != nil {
        fmt.Println("Error creating server:", err)
        return
    }

    fmt.Println("Server started on", server.listener.Addr())

    // Goroutine 1: Application logic (single-threaded echo server)
    go server.applicationLogic()

    // Goroutine 2: Proxy - accepts new connections and messages
    go server.proxy()

    // Block forever
    select {}
}
