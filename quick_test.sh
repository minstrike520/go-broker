#!/bin/bash

# Quick manual test - just starts the brokers and subscriber
# You can manually test publishing and killing primary

echo "=== Quick Manual Test Setup ==="
echo ""

# Start Primary broker
echo "Starting Primary broker on port 8080..."
gnome-terminal -- bash -c "go run ./cmd/server/main.go 8080 localhost:8081; exec bash" 2>/dev/null || \
xterm -e "go run ./cmd/server/main.go 8080 localhost:8081" 2>/dev/null || \
(go run ./cmd/server/main.go 8080 localhost:8081 &)

sleep 2

# Start Backup broker
echo "Starting Backup broker on port 8081..."
gnome-terminal -- bash -c "go run ./cmd/backup/main.go 8081 localhost:8080; exec bash" 2>/dev/null || \
xterm -e "go run ./cmd/backup/main.go 8081 localhost:8080" 2>/dev/null || \
(go run ./cmd/backup/main.go 8081 localhost:8080 &)

sleep 2

# Start subscriber
echo "Starting subscriber for topic 'topicC'..."
gnome-terminal -- bash -c "go run ./cmd/subscriber/main.go topicC localhost:8080 localhost:8081; exec bash" 2>/dev/null || \
xterm -e "go run ./cmd/subscriber/main.go topicC localhost:8080 localhost:8081" 2>/dev/null || \
(go run ./cmd/subscriber/main.go topicC localhost:8080 localhost:8081 &)

sleep 1

echo ""
echo "✓ All components started!"
echo ""
echo "To test manually:"
echo ""
echo "1. Send messages with test publisher (10 Hz):"
echo "   go run ./cmd/test_publisher/main.go topicC localhost:8080 localhost:8081"
echo ""
echo "2. Or send single messages:"
echo "   go run ./cmd/publisher/main.go topicC \"Hello\" localhost:8080 localhost:8081"
echo ""
echo "3. Kill the Primary broker process to test failover"
echo ""
echo "4. Watch the subscriber terminal to see messages continue from Backup"
echo ""
