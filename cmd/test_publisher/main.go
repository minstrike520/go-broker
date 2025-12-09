package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

type Message struct {
	seqNum  int
	topic   string
	payload string
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run cmd/test_publisher/main.go <topic> <primary-host:port> <backup-host:port>")
		return
	}

	topic := os.Args[1]
	primaryAddr := os.Args[2]
	backupAddr := os.Args[3]

	// Keep last 5 messages
	recentMessages := make([]Message, 0, 5)
	seqNum := 1
	usePrimary := true
	ticker := time.NewTicker(100 * time.Millisecond) // 10 Hz = 100ms interval
	defer ticker.Stop()

	fmt.Printf("Starting test publisher: topic=%s, primary=%s, backup=%s\n", topic, primaryAddr, backupAddr)
	fmt.Println("Sending messages at 10 Hz (every 100ms)")

	for range ticker.C {
		payload := strconv.Itoa(seqNum)
		msg := Message{
			seqNum:  seqNum,
			topic:   topic,
			payload: payload,
		}

		// Add to recent messages buffer
		recentMessages = append(recentMessages, msg)
		if len(recentMessages) > 5 {
			recentMessages = recentMessages[1:] // Keep only last 5
		}

		var success bool
		if usePrimary {
			success = sendMessageWithAck(topic, payload, primaryAddr, true)
			if !success {
				fmt.Println("Primary failed! Resending last 5 messages to Backup...")
				usePrimary = false

				// Resend last 5 messages to Backup
				for _, oldMsg := range recentMessages {
					sendMessageWithAck(topic, oldMsg.payload, backupAddr, false)
					time.Sleep(10 * time.Millisecond)
				}
			}
		} else {
			// Already failed over to backup
			sendMessageWithAck(topic, payload, backupAddr, false)
		}

		seqNum++
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

	// Send PUBLISH packet: PUBLISH|TOPIC|MESSAGE
	publishPacket := fmt.Sprintf("PUBLISH|%s|%s\n", topic, message)
	_, err = conn.Write([]byte(publishPacket))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return false
	}

	if !waitForAck {
		fmt.Printf("Published to backup: %s\n", message)
		return true
	}

	// Wait for ACK with 500ms timeout
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')

	if err != nil {
		fmt.Printf("Timeout waiting for ACK (message: %s)\n", message)
		return false
	}

	if response == "ACK\n" {
		fmt.Printf("Published and ACKed: %s\n", message)
		return true
	}

	fmt.Println("No ACK received")
	return false
}
