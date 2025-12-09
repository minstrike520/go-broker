package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run cmd/publisher/main.go <topic> <message> [broker-host:port]")
		return
	}

	topic := os.Args[1]
	message := os.Args[2]
	brokerAddr := "localhost:8080"
	if len(os.Args) > 3 {
		brokerAddr = os.Args[3]
	}

	// Connect to the broker
	conn, err := net.Dial("tcp", brokerAddr)
	if err != nil {
		fmt.Println("Error connecting to broker:", err)
		return
	}
	defer conn.Close()

	fmt.Printf("Connected to broker at %s. Publishing to topic: %s\n", brokerAddr, topic)

	// Send PUBLISH packet: PUBLISH|TOPIC|MESSAGE
	publishPacket := fmt.Sprintf("PUBLISH|%s|%s\n", topic, message)
	_, err = conn.Write([]byte(publishPacket))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Printf("Message published: %s\n", message)
}
