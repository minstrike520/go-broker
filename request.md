# Broker implementation
## Basic MQTT structure
- subscribers - a broker - publishers

## Transformation from current state
- Currently the structure is client-server.
- We are going to transform Server into Broker.
- Client now diverges into Subscriber and Publisher.

## Specifications aside from MQTT
- Control packets: **PUBLISH** and **SUBSCRIBE**. Unlike MQTT, no need to implement **CONNECT**, **PUBACK**, etc. The broker simply reacts based on the control packet type.
- The packet header includes both control packet type and topic name. The packet payload includes the message to be sent. A **SUBSCRIBE** packet can have no payload.
- The broker will take the topic information from each **SUBSCRIBE** packet and add the topic to a hash table. The hash table will then be used in the future for topic matching.
- The broker will take the topic information from each **PUBLISH** packet and look up the constructed hash table, to find all subscribers that have subscribed to the topic. The broker will forward the message to each relevant subscriber.
- Each publisher sends a message and disconnects. Each subscriber connects and keeps receiving messages unless we press CTRL-C and kill the process. The subscriber will print on the screen each message received.

Your implementation must support the following step-by-step application scenario (i.e. this is a possible usecase):
1. A subscriber connects to the broker with its subscription request for topic `topicA`.
2. Another subscriber connects to the broker with its subscription request for topic `topicB`.
3. A publisher connects to the broker with topic `topicA` and message `'Hello'`.
4. The first subscriber should receive the `'Hello'` message. The second subscriber should not receive the message because the topic name does not match.
5. A third subscriber connects to the broker with its subscription request for topic topicA.
6. A publisher connects to the broker with topic `topicA` and message `'Bye'`. Now, this message should be received by both the first and the third subscribers.