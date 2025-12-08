#!/bin/bash

echo "Testing Broker Implementation"
echo "=============================="
echo ""
echo "Instructions:"
echo "1. Start the broker: go run cmd/server/main.go"
echo "2. In separate terminals, run subscribers:"
echo "   Terminal 2: go run cmd/subscriber/main.go topicA"
echo "   Terminal 3: go run cmd/subscriber/main.go topicB"
echo "   Terminal 4: go run cmd/subscriber/main.go topicA"
echo "3. Publish messages:"
echo "   go run cmd/publisher/main.go topicA Hello"
echo "   go run cmd/publisher/main.go topicA Bye"
echo ""
echo "Expected behavior:"
echo "- First 'Hello' message goes to first subscriber (topicA)"
echo "- 'Bye' message goes to first and third subscribers (both subscribed to topicA)"
echo "- Second subscriber (topicB) receives no messages"

# Test script for the broker implementation

echo "Testing broker implementation..."
echo ""

# Test commands
test_messages=(
    "hello"
    "world"
    "echo test"
    "goodbye"
)

# Send test messages to the server
(
    sleep 1
    for msg in "${test_messages[@]}"; do
        echo "$msg"
        sleep 0.5
    done
) | go run cmd/client/main.go

echo ""
echo "Test completed!"
