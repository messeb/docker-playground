# RabbitMQ Producer / Consumer

A minimal work-queue example using RabbitMQ with two Python services. The producer publishes numbered messages; the consumer processes them one at a time as they arrive. Watch the full flow live in your terminal.

## Architecture

```text
Producer ──► RabbitMQ (queue: messages) ──► Consumer
              │
              └─► Management UI (http://localhost:15672)
```

- **Producer** sends `MESSAGE_COUNT` messages to the queue, one per second, then exits
- **Consumer** runs continuously, printing each message the moment it is delivered
- **RabbitMQ** persists the queue on disk — messages survive a broker restart

## Project structure

```text
rabbitmq-producer-consumer/
├── compose.yml
├── Makefile
├── producer/
│   ├── Dockerfile
│   └── producer.py   # connects, declares queue, publishes N messages
└── consumer/
    ├── Dockerfile
    └── consumer.py   # connects, declares queue, waits and acks messages
```

## Quick start

```bash
make demo
```

Starts the broker in the background, then runs producer and consumer in parallel with their logs interleaved live.

## What you will see

Producer and consumer logs stream side by side, prefixed by service name:

```
rmq-consumer  | [10:42:00] [consumer] Queue 'messages' ready  (prefetch=1)
rmq-consumer  | [10:42:00] [consumer] Waiting for messages  (CTRL+C to stop)
rmq-producer  | [10:42:00] [producer] Queue 'messages' ready  (durable=True)
rmq-producer  | [10:42:00] [producer] Sending 20 messages  delay=1.0s
rmq-producer  |   [ 1/20]  10:42:00  →  Message 1 of 20
rmq-consumer  |   [  1]  10:42:00  ←  Message 1 of 20
rmq-producer  |   [ 2/20]  10:42:01  →  Message 2 of 20
rmq-consumer  |   [  2]  10:42:01  ←  Message 2 of 20
rmq-producer  |   [ 3/20]  10:42:02  →  Message 3 of 20
rmq-consumer  |   [  3]  10:42:02  ←  Message 3 of 20
  ...
rmq-producer  | [10:42:20] [producer] All 20 messages sent. Connection closed.
rmq-consumer  | [10:42:20] [consumer] Waiting for messages  (CTRL+C to stop)
```

The `→` and `←` arrows show the message flow through the broker. Producer exits after all messages are sent; consumer keeps running until CTRL+C.

## Key concepts

### Durable queues and persistent messages

```python
# declare queue as durable — definition survives broker restart
channel.queue_declare(queue=QUEUE, durable=True)

# mark each message as persistent — written to disk before broker confirms
pika.BasicProperties(delivery_mode=2)
```

A **durable queue** survives a RabbitMQ restart — its definition is written to disk. A **persistent message** is also written to disk before RabbitMQ confirms receipt. Together they guarantee no messages are lost if the broker crashes between publish and consume.

### Manual acknowledgement

```python
# ack only after the message is fully processed
ch.basic_ack(delivery_tag=method.delivery_tag)
```

RabbitMQ keeps an unacknowledged message in memory until the consumer sends an `ack`. If the consumer crashes before acking, RabbitMQ requeues the message automatically — no message is lost.

### Fair dispatch with prefetch

```python
channel.basic_qos(prefetch_count=1)
```

With `prefetch_count=1`, RabbitMQ never delivers more than one unacknowledged message to a consumer at a time. When scaled to multiple consumers, work is distributed evenly — a slow consumer doesn't accumulate a backlog while fast ones idle.

## Management UI

Open [http://localhost:15672](http://localhost:15672) while the demo runs (user: `guest`, pass: `guest`).

| Tab | What to look for |
| --- | --- |
| **Queues** → `messages` | Message depth, publish/deliver rate, number of consumers |
| **Connections** | One entry per running producer/consumer |
| **Overview** | Global publish and deliver rates as live graphs |

## Configuration

Edit environment variables in `compose.yml`:

| Variable | Default | Description |
| --- | --- | --- |
| `MESSAGE_COUNT` | `20` | Number of messages the producer sends |
| `MESSAGE_DELAY` | `1` | Seconds between messages |
| `QUEUE_NAME` | `messages` | Queue name (must match in both services) |

## Experiments

**Scale to multiple consumers** — messages are distributed round-robin:
```bash
docker compose up --build -d rabbitmq
docker compose up --build -d --scale consumer=3
docker compose up --build producer
docker compose logs -f consumer
```

**Fill the queue before starting the consumer** — set `MESSAGE_DELAY=0` and start the producer without the consumer. Watch the queue depth build up in the management UI under Queues → `messages`, then drain when you start the consumer.

**Simulate a slow consumer** — add `time.sleep(2)` before `ch.basic_ack(...)` in `consumer.py`. With `prefetch_count=1`, queue depth grows in the management UI while the consumer works through messages one by one.

## Usage

| Command | Description |
| --- | --- |
| `make demo` | Build images, start broker + consumer, run producer, follow logs |
| `make logs` | Follow live logs for all services |
| `make open-ui` | Open RabbitMQ management UI in the browser |
| `make clean` | Stop containers and remove built images |

## Stop

```bash
make clean
```
