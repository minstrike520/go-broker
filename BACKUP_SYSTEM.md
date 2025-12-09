# Broker Backup and Failover System

## Overview

This implementation adds fault tolerance to the broker system through a Primary-Backup architecture with automatic failover.

## New Features

### 1. Pseudo Computing
- Each message undergoes pseudo computing (50-150ms uniform distribution) before forwarding
- Simulates edge computing workload

### 2. Primary-Backup Architecture
- **Primary Broker**: Main broker handling all messages
- **Backup Broker**: Standby broker that monitors Primary and takes over on failure

### 3. Replication Protocol
- Primary replicates messages to Backup before processing
- Primary sends CLEAR command to Backup after successful processing
- Ensures Backup has recent message state

### 4. Alive-Check Mechanism
- Backup polls Primary every 1 second with PING
- Primary responds with PONG
- If Primary fails to respond, Backup takes over

### 5. Publisher Acknowledgment
- Primary sends ACK to publisher for each message
- Publisher waits 500ms for ACK
- If no ACK, publisher assumes Primary crashed

### 6. Automatic Failover
- Publisher maintains last 5 message copies
- On Primary failure, publisher resends last 5 messages to Backup
- Future messages sent to Backup

## Programs

### Primary Broker (`cmd/server/main.go`)
```bash
go run cmd/server/main.go <port> <backup-host:port>
```
Example:
```bash
go run cmd/server/main.go 8080 localhost:8081
```

### Backup Broker (`cmd/backup/main.go`)
```bash
go run cmd/backup/main.go <port> <primary-host:port>
```
Example:
```bash
go run cmd/backup/main.go 8081 localhost:8080
```

### Subscriber (connects to both brokers)
```bash
go run cmd/subscriber/main.go <topic> <primary-host:port> <backup-host:port>
```
Example:
```bash
go run cmd/subscriber/main.go topicC localhost:8080 localhost:8081
```

### Publisher (with failover support)
```bash
go run cmd/publisher/main.go <topic> <message> <primary-host:port> <backup-host:port>
```
Example:
```bash
go run cmd/publisher/main.go topicC "Hello" localhost:8080 localhost:8081
```

### Test Publisher (10 Hz with sequence numbers)
```bash
go run cmd/test_publisher/main.go <topic> <primary-host:port> <backup-host:port>
```
Example:
```bash
go run cmd/test_publisher/main.go topicC localhost:8080 localhost:8081
```

## Test Scenario

The `test_backup.sh` script demonstrates the failover mechanism:

1. Start Primary broker on port 8080
2. Start Backup broker on port 8081
3. Start subscriber connecting to both brokers for topic `topicC`
4. Start test publisher sending at 10 Hz (sequence numbers 1, 2, 3, ...)
5. After 30 seconds, kill Primary broker
6. Observe subscriber continues receiving messages from Backup

### Running the Test

```bash
./test_backup.sh
```

The script will:
- Start all necessary processes
- Wait 30 seconds
- Kill Primary broker
- Continue for 10 more seconds
- Clean up all processes

### Expected Results

Check `subscriber.log` to observe:
- Messages 1-300 (approximately) from Primary
- Brief gap during failover detection
- Last 5 messages resent from publisher to Backup
- Messages continue from Backup (301, 302, ...)

## Protocol Messages

### PUBLISH
Format: `PUBLISH|<topic>|<payload>`
- Sent by publisher to broker
- Triggers replication (Primary) or direct processing (Backup when Primary is down)

### REPLICATE
Format: `REPLICATE|<topic>|<payload>`
- Sent by Primary to Backup
- Stores message in Backup's buffer

### CLEAR
Format: `CLEAR|<topic>|<payload>`
- Sent by Primary to Backup after processing
- Removes message from Backup's buffer

### ACK
Format: `ACK`
- Sent by Primary to publisher
- Confirms message receipt

### PING/PONG
Format: `PING` / `PONG`
- Sent by Backup to Primary for alive-check
- Primary responds with PONG if alive

## Architecture Details

### Primary Broker Flow
1. Receive PUBLISH from publisher
2. Replicate to Backup (REPLICATE)
3. Send ACK to publisher
4. Perform pseudo computing (50-150ms)
5. Forward to subscribers
6. Clear from Backup (CLEAR)

### Backup Broker Flow (Primary Alive)
1. Receive REPLICATE messages
2. Store in buffer
3. Receive CLEAR messages
4. Remove from buffer
5. Respond to PING with PONG

### Backup Broker Flow (Primary Down)
1. Detect Primary failure (no PONG)
2. Process all buffered replicated messages
3. Accept new PUBLISH messages directly
4. Perform pseudo computing and forward

### Publisher Flow
1. Send PUBLISH to Primary
2. Wait 500ms for ACK
3. If ACK received: continue
4. If timeout: switch to Backup, resend last 5 messages
5. Send all future messages to Backup

## Notes

- Subscriber connects to both brokers to receive messages from whichever is active
- Message deduplication is not implemented; subscribers may receive duplicates during failover
- The 50-150ms computing delay simulates edge processing workload
- Publisher's 5-message buffer ensures minimal message loss during failover
