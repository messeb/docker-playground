"""
Consumer — subscribes to a RabbitMQ queue and processes messages as they arrive.

The consumer runs indefinitely until stopped (CTRL+C / docker compose down).
It acknowledges each message only after processing it — if the consumer crashes
mid-processing, RabbitMQ requeues the message for another consumer.

Key concepts shown here:
  - manual acknowledgement (basic_ack): message stays in the queue until the
    consumer explicitly confirms it was processed
  - prefetch_count=1: RabbitMQ sends the consumer at most 1 unacknowledged
    message at a time — prevents one slow consumer from hoarding the queue
  - durable queue declaration: must match the producer's declaration

Environment variables:
  RABBITMQ_HOST   RabbitMQ hostname  (default: localhost)
  RABBITMQ_USER   Username           (default: guest)
  RABBITMQ_PASS   Password           (default: guest)
  QUEUE_NAME      Queue to consume   (default: messages)
"""

import os
import sys
import datetime
import pika

HOST  = os.getenv("RABBITMQ_HOST", "localhost")
USER  = os.getenv("RABBITMQ_USER", "guest")
PASS  = os.getenv("RABBITMQ_PASS", "guest")
QUEUE = os.getenv("QUEUE_NAME",    "messages")

received = 0  # running count for display


def now():
    return datetime.datetime.now().strftime("%H:%M:%S")


# ── Connect to broker ─────────────────────────────────────────────────────────
print(f"[{now()}] [consumer] Connecting to RabbitMQ at {HOST}:5672 …")
try:
    connection = pika.BlockingConnection(
        pika.ConnectionParameters(
            host=HOST,
            credentials=pika.PlainCredentials(USER, PASS),
        )
    )
except pika.exceptions.AMQPConnectionError as exc:
    print(f"[{now()}] [consumer] ERROR: {exc}", file=sys.stderr)
    sys.exit(1)

channel = connection.channel()

# ── Declare queue ─────────────────────────────────────────────────────────────
# Must match the producer's declaration (same name, same durable flag).
# Declaring an already-existing queue with identical parameters is a no-op.
channel.queue_declare(queue=QUEUE, durable=True)

# prefetch_count=1: don't dispatch a new message until this consumer has
# acknowledged the previous one — enables fair dispatch across multiple consumers.
channel.basic_qos(prefetch_count=1)


# ── Message handler ───────────────────────────────────────────────────────────
def on_message(ch, method, properties, body):
    """Called by pika for every message delivered from the queue."""
    global received
    received += 1

    print(f"  [{received:>3}]  {now()}  ←  {body.decode()}")

    # Acknowledge the message — RabbitMQ removes it from the queue.
    # Without this ack, the message would be requeued when the connection closes.
    ch.basic_ack(delivery_tag=method.delivery_tag)


# ── Start consuming ───────────────────────────────────────────────────────────
channel.basic_consume(queue=QUEUE, on_message_callback=on_message)

print(f"[{now()}] [consumer] Queue '{QUEUE}' ready  (prefetch=1)")
print(f"[{now()}] [consumer] Waiting for messages  (CTRL+C to stop)\n")

try:
    channel.start_consuming()
except KeyboardInterrupt:
    channel.stop_consuming()
    connection.close()
    print(f"\n[{now()}] [consumer] Stopped. {received} messages received.")
