#!/bin/bash

# Demo script showing basic usage of all components

echo "================================================"
echo "     Broker Backup System - Usage Demo"
echo "================================================"
echo ""
echo "This demo shows how to use each component."
echo "Follow the instructions to see the system in action."
echo ""

show_usage() {
    echo "--- $1 ---"
    echo "$2"
    echo ""
}

show_usage "1. PRIMARY BROKER" \
"go run ./cmd/server/main.go <port> <backup-host:port>

Example:
  go run ./cmd/server/main.go 8080 localhost:8081

This starts the Primary broker which:
- Accepts client connections on specified port
- Replicates messages to Backup
- Sends ACKs to publishers
- Performs pseudo computing (50-150ms)
- Forwards messages to subscribers"

show_usage "2. BACKUP BROKER" \
"go run ./cmd/backup/main.go <port> <primary-host:port>

Example:
  go run ./cmd/backup/main.go 8081 localhost:8080

This starts the Backup broker which:
- Monitors Primary with periodic PING (every 1s)
- Stores replicated messages
- Takes over when Primary fails
- Performs same processing as Primary"

show_usage "3. SUBSCRIBER" \
"go run ./cmd/subscriber/main.go <topic> <primary-host:port> <backup-host:port>

Example:
  go run ./cmd/subscriber/main.go topicC localhost:8080 localhost:8081

Connects to both brokers and receives messages from whichever is active.
Messages are labeled [Primary] or [Backup]."

show_usage "4. SINGLE MESSAGE PUBLISHER" \
"go run ./cmd/publisher/main.go <topic> <message> <primary-host:port> <backup-host:port>

Example:
  go run ./cmd/publisher/main.go topicC 'Hello World' localhost:8080 localhost:8081

Sends a single message with automatic failover:
- Tries Primary first (waits for ACK)
- Falls back to Backup on timeout"

show_usage "5. TEST PUBLISHER (10 Hz)" \
"go run ./cmd/test_publisher/main.go <topic> <primary-host:port> <backup-host:port>

Example:
  go run ./cmd/test_publisher/main.go topicC localhost:8080 localhost:8081

Continuously sends sequence numbers at 10 Hz:
- Messages: 1, 2, 3, 4, ...
- Maintains last 5 message buffer
- Automatic failover with message resend"

echo ""
echo "================================================"
echo "           TESTING SCENARIOS"
echo "================================================"
echo ""

show_usage "SCENARIO 1: Basic Functionality" \
"Terminal 1: go run ./cmd/server/main.go 8080 localhost:8081
Terminal 2: go run ./cmd/backup/main.go 8081 localhost:8080
Terminal 3: go run ./cmd/subscriber/main.go topicC localhost:8080 localhost:8081
Terminal 4: go run ./cmd/publisher/main.go topicC 'Test' localhost:8080 localhost:8081

Expected: Subscriber receives 'Test' from Primary"

show_usage "SCENARIO 2: Continuous Publishing" \
"Terminal 1: go run ./cmd/server/main.go 8080 localhost:8081
Terminal 2: go run ./cmd/backup/main.go 8081 localhost:8080
Terminal 3: go run ./cmd/subscriber/main.go topicC localhost:8080 localhost:8081
Terminal 4: go run ./cmd/test_publisher/main.go topicC localhost:8080 localhost:8081

Expected: Subscriber receives sequence 1, 2, 3, 4, ... from Primary"

show_usage "SCENARIO 3: Failover Test (MAIN TEST)" \
"Terminal 1: go run ./cmd/server/main.go 8080 localhost:8081
Terminal 2: go run ./cmd/backup/main.go 8081 localhost:8080
Terminal 3: go run ./cmd/subscriber/main.go topicC localhost:8080 localhost:8081
Terminal 4: go run ./cmd/test_publisher/main.go topicC localhost:8080 localhost:8081

Wait 30 seconds...
Kill Terminal 1 (Primary) with Ctrl+C

Expected:
- Messages 1-300 from Primary
- Brief gap
- Last 5 messages resent
- Messages continue from Backup"

show_usage "AUTOMATED TEST" \
"./test_backup.sh

Runs the complete Scenario 3 automatically:
- Starts all components
- Waits 30 seconds
- Kills Primary
- Observes for 10 more seconds
- Saves logs to *.log files"

show_usage "QUICK MANUAL TEST" \
"./quick_test.sh

Starts Primary, Backup, and Subscriber in separate terminals.
You can then manually run publishers and test failover."

echo ""
echo "================================================"
echo "              TIPS & NOTES"
echo "================================================"
echo ""
echo "• Computing Delay: Each message takes 50-150ms to process"
echo "• ACK Timeout: Publisher waits 500ms for ACK from Primary"
echo "• Alive-Check: Backup pings Primary every 1 second"
echo "• Message Buffer: Publisher keeps last 5 messages"
echo "• At 10 Hz: Expect ~300 messages in 30 seconds"
echo "• Failover: Automatic when Primary doesn't respond"
echo "• Logs: Check *.log files after test_backup.sh"
echo ""
echo "================================================"
echo "            MONITORING TIPS"
echo "================================================"
echo ""
echo "Watch subscriber in real-time:"
echo "  tail -f subscriber.log"
echo ""
echo "Watch publisher in real-time:"
echo "  tail -f publisher.log"
echo ""
echo "Count messages from each broker:"
echo "  grep '\\[Primary\\]' subscriber.log | wc -l"
echo "  grep '\\[Backup\\]' subscriber.log | wc -l"
echo ""
echo "See message sequence:"
echo "  grep 'Received:' subscriber.log | tail -20"
echo ""
echo "================================================"
