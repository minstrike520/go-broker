package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

type Message struct {
	topic   string
	payload string
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run cmd/publisher/main.go <topic> <message> <primary-host:port> <backup-host:port>")
		return
	}

	topic := os.Args[1]
	message := os.Args[2]
	primaryAddr := os.Args[3]
	backupAddr := os.Args[4]

	// Send message to Primary
	success := sendMessageWithAck(topic, message, primaryAddr, true)

	if !success {
		fmt.Println("Primary failed, switching to backup...")
		sendMessageWithAck(topic, message, backupAddr, false)
	}
}

func sendMessageWithAck(topic, message, brokerAddr string, waitForAck bool) bool {
	// Connect to the broker
	conn, err := net.Dial("tcp", brokerAddr)
	if err != nil {
		fmt.Println("Error connecting to broker:", err)
		return false
	}
	defer conn.Close()

	fmt.Printf("Connected to broker at %s. Publishing to topic: %s\n", brokerAddr, topic)

	// Send PUBLISH packet: PUBLISH|TOPIC|MESSAGE
	publishPacket := fmt.Sprintf("PUBLISH|%s|%s\n", topic, message)
	_, err = conn.Write([]byte(publishPacket))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return false
	}

	if !waitForAck {
		fmt.Printf("Message published to backup: %s\n", message)
		return true
	}

	// Wait for ACK with 500ms timeout
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')

	if err != nil {
		fmt.Println("Timeout waiting for ACK from primary")
		return false
	}

	if response == "ACK\n" {
		fmt.Printf("Message published and acknowledged: %s\n", message)
		return true
	}

	fmt.Println("No ACK received")
	return false
}
