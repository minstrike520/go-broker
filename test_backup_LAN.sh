#!/bin/bash

# Test script for broker backup mechanics
# This script demonstrates the failover scenario described in the requirements

echo "=== Broker Backup Test ==="
echo ""
echo "This test will:"
echo "1. Start Primary broker on port 8080"
echo "2. Start Backup broker on port 8081"
echo "3. Start subscriber connecting to both brokers"
echo "4. Start test publisher sending at 10 Hz"
echo "5. After 30 seconds, kill the Primary broker"
echo "6. Observe failover to Backup"
echo ""
echo "Press Enter to start..."
read

IP1="10.243.248.66" # pad
IP2="10.243.248.49" # rpi
IP_LOCAL="10.243.248.195"

ssh1="ssh $IP1 -p 8022"
ssh2="ssh $IP2"

set_tab_name() {
    echo -ne "\033]30;$1\007"
}

term() {
    konsole --hold -e bash -c "echo -ne \"\033]30;$2\007\"; $1; exec bash" &
}


# Start Primary broker
echo "Starting Primary broker on port 8080..."
term "$ssh1 \"cd ~/repos/go-broker; go run ./cmd/server/main.go 8080 $IP2:8081\"" "PRIMARY BROKER"
PRIMARY_PID=$!
echo "Primary PID: $PRIMARY_PID"
sleep 4

# Start Backup broker
echo "Starting Backup broker on port 8081..."
term "$ssh2 \"cd ~/go-broker; go run ./cmd/backup/main.go 8081 $IP1:8080\"" "BACKUP BROKER"
BACKUP_PID=$!
echo "Backup PID: $BACKUP_PID"
sleep 4

# Start subscriber
echo "Starting subscriber for topic 'topicC'..."
term "go run ./cmd/subscriber/main.go topicC $IP1:8080 $IP2:8081" "SUBSCRIBER"
SUBSCRIBER_PID=$!
echo "Subscriber PID: $SUBSCRIBER_PID"
sleep 4

# Start test publisher
echo "Starting test publisher (10 Hz)..."
term "go run ./cmd/test_publisher/main.go topicC $IP1:8080 $IP2:8081" "PUBLISHER"
PUBLISHER_PID=$!
echo "Publisher PID: $PUBLISHER_PID"

echo ""
echo "All processes started. Waiting 30 seconds..."
echo "You can tail the logs in another terminal:"
echo "  tail -f subscriber.log"
echo "  tail -f publisher.log"
echo ""

# Wait 30 seconds
for i in {30..1}; do
    echo -ne "Time until Primary kill: $i seconds\r"
    sleep 1
done
echo ""

# Kill Primary
echo "🔥 KILLING PRIMARY BROKER (PID: $PRIMARY_PID)..."
kill $PRIMARY_PID
echo "Primary broker killed!"

echo ""
echo "Backup should take over. Observing for 10 more seconds..."

for i in {10..1}; do
    # echo -ne "Time until Publisher kill: $i seconds\r"
    sleep 1
done

# kill $PUBLISHER_PID

echo ""
echo "=== Test Complete ==="
echo ""
echo "Check the logs to verify:"
echo "1. Messages 1-300 received from Primary"
echo "2. Brief gap during failover"
echo "3. Messages continue from Backup (including resent last 5)"
echo ""

echo "Done! Press Enter to stop..."
read

kill $BACKUP_PID $SUBSCRIBER_PID $PUBLISHER_PID 2>/dev/null
echo "Cleaning up..."
wait 2>/dev/null
