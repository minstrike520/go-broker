#!/bin/bash

# Quick manual test - just starts the brokers and subscriber
# You can manually test publishing and killing primary

echo "=== Quick Manual Test Setup ==="
echo ""

A=""

term() {
    konsole --hold -e bash -c "$1; exec bash" &
}

# Start Primary broker
echo "Starting Primary broker on port 8080..."
term "go run ./cmd/server/main.go 8080 localhost:8081"
A="$A $!"
sleep 1

# Start Backup broker
echo "Starting Backup broker on port 8081..."
konsole --hold -e bash -c "go run ./cmd/backup/main.go 8081 localhost:8080; exec bash" 2>/dev/null&
A="$A $!"
sleep 1


# Start subscriber
echo "Starting subscriber for topic 'topicC'..."
konsole --hold -e bash -c "go run ./cmd/subscriber/main.go topicC localhost:8080 localhost:8081; exec bash" 2>/dev/null&
A="$A $!"
sleep 1

konsole --hold -e bash -c "go run ./cmd/publisher/main.go topicC content localhost:8080 localhost:8081; exec bash" 2>/dev/null&
A="$A $!"

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

echo "To kill all windows:"

echo "   kill $A 2>/dev/null"