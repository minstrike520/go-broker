#!/bin/bash

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
