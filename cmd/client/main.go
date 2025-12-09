package main

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "strings"
)

func main() {
    serverAddr := "localhost:8080"
    if len(os.Args) > 1 {
        serverAddr = os.Args[1]
    }

    fmt.Printf("Usage: go run cmd/client/main.go [host:port]\n")
    fmt.Printf("Connecting to server at %s\n", serverAddr)

    // Connect to the server
    conn, err := net.Dial("tcp", serverAddr)
    if err != nil {
        fmt.Println("Error connecting to server:", err)
        return
    }
    defer conn.Close()

    fmt.Println("Connected to server. Type messages (type 'goodbye' to quit):")

    // Read user input and send to server
    go func() {
        scanner := bufio.NewScanner(os.Stdin)
        for scanner.Scan() {
            text := scanner.Text()
            _, err := conn.Write([]byte(text + "\n"))
            if err != nil {
                fmt.Println("Error sending message:", err)
                return
            }

            if strings.ToLower(strings.TrimSpace(text)) == "goodbye" {
                return
            }
        }
    }()

    // Read responses from server
    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        response := scanner.Text()
        fmt.Println("Server:", response)
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Connection closed")
    }
}
