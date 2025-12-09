package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
)

func subscribeToBroker(topic, brokerAddr, brokerName string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Connect to the broker
	conn, err := net.Dial("tcp", brokerAddr)
	if err != nil {
		fmt.Printf("[%s] Error connecting to broker: %v\n", brokerName, err)
		return
	}
	defer conn.Close()

	fmt.Printf("[%s] Connected to broker at %s. Subscribing to topic: %s\n", brokerName, brokerAddr, topic)

	// Send SUBSCRIBE packet: SUBSCRIBE|TOPIC
	subscribePacket := fmt.Sprintf("SUBSCRIBE|%s\n", topic)
	_, err = conn.Write([]byte(subscribePacket))
	if err != nil {
		fmt.Printf("[%s] Error sending subscription: %v\n", brokerName, err)
		return
	}

	fmt.Printf("[%s] Waiting for messages...\n", brokerName)

	// Keep receiving messages from broker
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		message := scanner.Text()
		fmt.Printf("[%s] Received: %s\n", brokerName, message)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("[%s] Connection closed: %v\n", brokerName, err)
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run cmd/subscriber/main.go <topic> <primary-host:port> <backup-host:port>")
		return
	}

	topic := os.Args[1]
	primaryAddr := os.Args[2]
	backupAddr := os.Args[3]

	var wg sync.WaitGroup

	// Subscribe to Primary
	wg.Add(1)
	go subscribeToBroker(topic, primaryAddr, "Primary", &wg)

	// Subscribe to Backup
	wg.Add(1)
	go subscribeToBroker(topic, backupAddr, "Backup", &wg)

	fmt.Println("Subscribed to both Primary and Backup brokers. Press CTRL-C to quit")

	wg.Wait()
}
