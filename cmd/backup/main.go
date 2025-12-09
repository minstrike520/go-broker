package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// PacketType represents the control packet type
type PacketType string

const (
	PUBLISH   PacketType = "PUBLISH"
	SUBSCRIBE PacketType = "SUBSCRIBE"
	REPLICATE PacketType = "REPLICATE"
	CLEAR     PacketType = "CLEAR"
	ACK       PacketType = "ACK"
	PING      PacketType = "PING"
	PONG      PacketType = "PONG"
)

// Packet represents a message packet with header and payload
type Packet struct {
	conn        net.Conn
	controlType PacketType
	topic       string
	payload     string
}

// Broker handles pub/sub with topic-based routing
type Broker struct {
	listener         net.Listener
	packets          chan Packet
	closeConns       chan net.Conn
	subscribers      map[string][]net.Conn // topic -> list of subscriber connections
	subscriberMu     sync.Mutex
	replicatedMsgs   map[string]bool // topic|payload for backup
	replicatedMsgsMu sync.Mutex
	primaryAddr      string
	isPrimary        bool
	primaryAlive     bool
	primaryAliveMu   sync.Mutex
}

// NewBackupBroker creates a new backup broker instance
func NewBackupBroker(addr string, primaryAddr string) (*Broker, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Broker{
		listener:       listener,
		packets:        make(chan Packet, 10),
		closeConns:     make(chan net.Conn, 10),
		subscribers:    make(map[string][]net.Conn),
		replicatedMsgs: make(map[string]bool),
		primaryAddr:    primaryAddr,
		isPrimary:      false,
		primaryAlive:   true,
	}, nil
}

// Start starts the backup broker
func (b *Broker) Start() {
	fmt.Println("Backup Broker started on", b.listener.Addr())
	fmt.Println("Primary broker is at", b.primaryAddr)

	// Start alive-check goroutine
	go b.aliveCheck()

	// Goroutine 1: Application logic
	go b.applicationLogic()

	// Goroutine 2: Proxy
	go b.proxy()

	// Block forever
	select {}
}

// aliveCheck periodically pings the primary to check if it's alive
func (b *Broker) aliveCheck() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		conn, err := net.DialTimeout("tcp", b.primaryAddr, 300*time.Millisecond)
		if err != nil {
			b.primaryAliveMu.Lock()
			if b.primaryAlive {
				fmt.Println("⚠️  PRIMARY IS DOWN! Taking over...")
				b.primaryAlive = false
				// Process all replicated messages
				go b.processReplicatedMessages()
			}
			b.primaryAliveMu.Unlock()
			continue
		}

		// Send PING
		conn.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
		_, err = conn.Write([]byte("PING||\n"))
		if err != nil {
			conn.Close()
			b.primaryAliveMu.Lock()
			if b.primaryAlive {
				fmt.Println("⚠️  PRIMARY IS DOWN! Taking over...")
				b.primaryAlive = false
				go b.processReplicatedMessages()
			}
			b.primaryAliveMu.Unlock()
			continue
		}

		// Wait for PONG
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		reader := bufio.NewReader(conn)
		response, err := reader.ReadString('\n')
		conn.Close()

		if err != nil || strings.TrimSpace(response) != "PONG" {
			b.primaryAliveMu.Lock()
			if b.primaryAlive {
				fmt.Println("⚠️  PRIMARY IS DOWN! Taking over...")
				b.primaryAlive = false
				go b.processReplicatedMessages()
			}
			b.primaryAliveMu.Unlock()
		} else {
			b.primaryAliveMu.Lock()
			if !b.primaryAlive {
				fmt.Println("✓ Primary is back online")
				b.primaryAlive = true
			}
			b.primaryAliveMu.Unlock()
		}
	}
}

// processReplicatedMessages processes all replicated messages when becoming active
func (b *Broker) processReplicatedMessages() {
	b.replicatedMsgsMu.Lock()
	defer b.replicatedMsgsMu.Unlock()

	fmt.Printf("Processing %d replicated messages...\n", len(b.replicatedMsgs))

	for key := range b.replicatedMsgs {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) == 2 {
			topic := parts[0]
			payload := parts[1]

			// Create a dummy packet for publishing
			packet := Packet{
				conn:        nil,
				controlType: PUBLISH,
				topic:       topic,
				payload:     payload,
			}
			b.handlePublishBackup(packet)
		}
	}

	// Clear all replicated messages
	b.replicatedMsgs = make(map[string]bool)
}

// applicationLogic handles the broker logic
func (b *Broker) applicationLogic() {
	for {
		select {
		case packet := <-b.packets:
			switch packet.controlType {
			case SUBSCRIBE:
				b.handleSubscribe(packet)
			case PUBLISH:
				b.primaryAliveMu.Lock()
				alive := b.primaryAlive
				b.primaryAliveMu.Unlock()

				if !alive {
					// Process immediately if primary is down
					b.handlePublishBackup(packet)
				}
			case REPLICATE:
				// Store replicated message
				b.replicatedMsgsMu.Lock()
				key := packet.topic + "|" + packet.payload
				b.replicatedMsgs[key] = true
				b.replicatedMsgsMu.Unlock()
				fmt.Printf("Replicated: %s -> %s\n", packet.topic, packet.payload)
			case CLEAR:
				// Clear message after primary processed it
				b.replicatedMsgsMu.Lock()
				key := packet.topic + "|" + packet.payload
				delete(b.replicatedMsgs, key)
				b.replicatedMsgsMu.Unlock()
				fmt.Printf("Cleared: %s -> %s\n", packet.topic, packet.payload)
			}
		case conn := <-b.closeConns:
			b.handleDisconnect(conn)
		}
	}
}

// handleSubscribe adds a subscriber to the topic
func (b *Broker) handleSubscribe(packet Packet) {
	b.subscriberMu.Lock()
	defer b.subscriberMu.Unlock()

	if b.subscribers[packet.topic] == nil {
		b.subscribers[packet.topic] = make([]net.Conn, 0)
	}
	b.subscribers[packet.topic] = append(b.subscribers[packet.topic], packet.conn)
	fmt.Printf("Subscriber added for topic '%s' from %s\n", packet.topic, packet.conn.RemoteAddr())
}

// handlePublishBackup forwards message when backup takes over
func (b *Broker) handlePublishBackup(packet Packet) {
	// Pseudo computing: 50-150ms uniform distribution
	computeTime := 50 + rand.Intn(101)
	fmt.Printf("Computing for %d ms...\n", computeTime)
	time.Sleep(time.Duration(computeTime) * time.Millisecond)

	b.subscriberMu.Lock()
	subscribers := b.subscribers[packet.topic]
	b.subscriberMu.Unlock()

	fmt.Printf("Publishing message to topic '%s': %s (subscribers: %d)\n", packet.topic, packet.payload, len(subscribers))

	for _, conn := range subscribers {
		message := fmt.Sprintf("%s\n", packet.payload)
		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Error writing to subscriber:", err)
			b.closeConns <- conn
		}
	}

	if packet.conn != nil {
		packet.conn.Close()
		fmt.Println("Publisher disconnected:", packet.conn.RemoteAddr())
	}
}

// handleDisconnect removes a connection from all topic subscriptions
func (b *Broker) handleDisconnect(conn net.Conn) {
	b.subscriberMu.Lock()
	defer b.subscriberMu.Unlock()

	for topic, subs := range b.subscribers {
		for i, sub := range subs {
			if sub == conn {
				b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
				fmt.Printf("Subscriber removed from topic '%s': %s\n", topic, conn.RemoteAddr())
				break
			}
		}
	}
	conn.Close()
}

// parsePacket parses the packet format: CONTROLTYPE|TOPIC|PAYLOAD
func parsePacket(line string, conn net.Conn) (Packet, error) {
	parts := strings.SplitN(line, "|", 3)
	if len(parts) < 2 {
		return Packet{}, fmt.Errorf("invalid packet format")
	}

	controlType := PacketType(strings.TrimSpace(parts[0]))
	topic := strings.TrimSpace(parts[1])
	payload := ""
	if len(parts) == 3 {
		payload = strings.TrimSpace(parts[2])
	}

	return Packet{
		conn:        conn,
		controlType: controlType,
		topic:       topic,
		payload:     payload,
	}, nil
}

// proxy accepts new connections and reads from all clients
func (b *Broker) proxy() {
	type connState struct {
		conn   net.Conn
		reader *bufio.Reader
	}

	connections := make(map[net.Conn]*connState)

	for {
		// Try to accept new connection
		b.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Millisecond))
		conn, err := b.listener.Accept()
		if err == nil {
			fmt.Println("New connection from:", conn.RemoteAddr())
			connections[conn] = &connState{
				conn:   conn,
				reader: bufio.NewReader(conn),
			}
		}

		// Read from all connections
		for conn, state := range connections {
			conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
			line, err := state.reader.ReadString('\n')

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if err != io.EOF {
					fmt.Println("Error reading from client:", err)
				}
				delete(connections, conn)
				b.closeConns <- conn
				continue
			}

			text := strings.TrimSpace(line)
			if text == "" {
				continue
			}

			fmt.Printf("Received from %s: %s\n", conn.RemoteAddr(), text)

			packet, err := parsePacket(text, conn)
			if err != nil {
				fmt.Println("Error parsing packet:", err)
				delete(connections, conn)
				b.closeConns <- conn
				continue
			}

			b.packets <- packet

			// Remove publisher connections after sending packet
			if packet.controlType == PUBLISH {
				delete(connections, conn)
			}
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run cmd/backup/main.go <port> <primary-host:port>")
		return
	}

	port := ":" + os.Args[1]
	primaryAddr := os.Args[2]

	broker, err := NewBackupBroker(port, primaryAddr)
	if err != nil {
		fmt.Println("Error creating backup broker:", err)
		return
	}

	fmt.Printf("Starting BACKUP broker on port %s (primary: %s)\n", port, primaryAddr)
	broker.Start()
}
