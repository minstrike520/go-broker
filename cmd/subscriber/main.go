package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/subscriber/main.go <topic> [broker-host:port]")
		return
	}

	topic := os.Args[1]
	brokerAddr := "localhost:8080"
	if len(os.Args) > 2 {
		brokerAddr = os.Args[2]
	}

	// Connect to the broker
	conn, err := net.Dial("tcp", brokerAddr)
	if err != nil {
		fmt.Println("Error connecting to broker:", err)
		return
	}
	defer conn.Close()

	fmt.Printf("Connected to broker at %s. Subscribing to topic: %s\n", brokerAddr, topic)

	// Send SUBSCRIBE packet: SUBSCRIBE|TOPIC
	subscribePacket := fmt.Sprintf("SUBSCRIBE|%s\n", topic)
	_, err = conn.Write([]byte(subscribePacket))
	if err != nil {
		fmt.Println("Error sending subscription:", err)
		return
	}

	fmt.Println("Waiting for messages... (Press CTRL-C to quit)")

	// Keep receiving messages from broker
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		message := scanner.Text()
		fmt.Printf("Received: %s\n", message)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Connection closed:", err)
	}
}
