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
	listener     net.Listener
	packets      chan Packet
	closeConns   chan net.Conn
	subscribers  map[string][]net.Conn // topic -> list of subscriber connections
	subscriberMu sync.Mutex
	backupAddr   string
	backupConn   net.Conn
	backupMu     sync.Mutex
	isPrimary    bool
}

// NewBroker creates a new broker instance
func NewBroker(addr string, backupAddr string, isPrimary bool) (*Broker, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Broker{
		listener:    listener,
		packets:     make(chan Packet, 10),
		closeConns:  make(chan net.Conn, 10),
		subscribers: make(map[string][]net.Conn),
		backupAddr:  backupAddr,
		isPrimary:   isPrimary,
	}, nil
}

// Start starts the broker with two goroutines
func (b *Broker) Start() {
	fmt.Println("Broker started on", b.listener.Addr())

	// Goroutine 1: Application logic (handle PUBLISH and SUBSCRIBE)
	go b.applicationLogic()

	// Goroutine 2: Proxy - accepts new connections and reads from all clients
	go b.proxy()

	// Block forever
	select {}
}

// applicationLogic handles the broker logic (routing messages to subscribers)
func (b *Broker) applicationLogic() {
	replicatedMessages := make(map[string]bool) // Store replicated messages (topic|payload)

	for {
		select {
		case packet := <-b.packets:
			switch packet.controlType {
			case SUBSCRIBE:
				b.handleSubscribe(packet)
			case PUBLISH:
				b.handlePublish(packet)
			case REPLICATE:
				// Backup receives replication
				if !b.isPrimary {
					key := packet.topic + "|" + packet.payload
					replicatedMessages[key] = true
					fmt.Printf("Replicated message: %s -> %s\n", packet.topic, packet.payload)
				}
			case CLEAR:
				// Backup clears message after Primary processed it
				if !b.isPrimary {
					key := packet.topic + "|" + packet.payload
					delete(replicatedMessages, key)
					fmt.Printf("Cleared replicated message: %s -> %s\n", packet.topic, packet.payload)
				}
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

// handlePublish forwards message to all subscribers of the topic
func (b *Broker) handlePublish(packet Packet) {
	// Pseudo computing: 50-150ms uniform distribution
	computeTime := 50 + rand.Intn(101) // 50 to 150 ms
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

	// If Primary, clear message from backup
	if b.isPrimary && b.backupConn != nil {
		clearPacket := fmt.Sprintf("CLEAR|%s|%s\n", packet.topic, packet.payload)
		b.backupMu.Lock()
		b.backupConn.Write([]byte(clearPacket))
		b.backupMu.Unlock()
	}

	// Publishers disconnect after sending (send ACK before closing if it's from publisher)
	packet.conn.Close()
	fmt.Println("Publisher disconnected:", packet.conn.RemoteAddr())
}

// handleDisconnect removes a connection from all topic subscriptions
func (b *Broker) handleDisconnect(conn net.Conn) {
	b.subscriberMu.Lock()
	defer b.subscriberMu.Unlock()

	for topic, subs := range b.subscribers {
		for i, sub := range subs {
			if sub == conn {
				// Remove this subscriber
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
		// Try to accept new connection (non-blocking with short timeout)
		b.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Millisecond))
		conn, err := b.listener.Accept()
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
				b.closeConns <- conn
				continue
			}

			// Got a packet
			text := strings.TrimSpace(line)
			if text == "" {
				continue
			}

			fmt.Printf("Received from %s: %s\n", conn.RemoteAddr(), text)

			// Parse and send packet to application logic
			packet, err := parsePacket(text, conn)
			if err != nil {
				fmt.Println("Error parsing packet:", err)
				delete(connections, conn)
				b.closeConns <- conn
				continue
			}

			// If Primary receives PUBLISH, replicate to backup first
			if b.isPrimary && packet.controlType == PUBLISH {
				if b.backupConn != nil {
					replicatePacket := fmt.Sprintf("REPLICATE|%s|%s\n", packet.topic, packet.payload)
					b.backupMu.Lock()
					_, err := b.backupConn.Write([]byte(replicatePacket))
					b.backupMu.Unlock()
					if err != nil {
						fmt.Println("Error replicating to backup:", err)
						b.backupConn = nil
					}
				}

				// Send ACK to publisher
				ackPacket := fmt.Sprintf("ACK\n")
				conn.Write([]byte(ackPacket))
			}

			// Handle PING from backup
			if packet.controlType == PING {
				pongPacket := fmt.Sprintf("PONG\n")
				conn.Write([]byte(pongPacket))
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
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/server/main.go <port> [backup-host:port]")
		fmt.Println("  If backup address is provided, this will be Primary broker")
		return
	}

	port := ":" + os.Args[1]
	backupAddr := ""
	isPrimary := false

	if len(os.Args) > 2 {
		backupAddr = os.Args[2]
		isPrimary = true
	}

	broker, err := NewBroker(port, backupAddr, isPrimary)
	if err != nil {
		fmt.Println("Error creating broker:", err)
		return
	}

	// If Primary, connect to Backup
	if isPrimary && backupAddr != "" {
		go func() {
			for {
				conn, err := net.Dial("tcp", backupAddr)
				if err != nil {
					fmt.Println("Failed to connect to backup, retrying in 2s:", err)
					time.Sleep(2 * time.Second)
					continue
				}
				fmt.Println("Connected to backup broker at", backupAddr)
				broker.backupMu.Lock()
				broker.backupConn = conn
				broker.backupMu.Unlock()
				break
			}
		}()
	}

	if isPrimary {
		fmt.Printf("Starting PRIMARY broker on port %s (backup: %s)\n", port, backupAddr)
	} else {
		fmt.Printf("Starting BACKUP broker on port %s\n", port)
	}

	broker.Start()
}
