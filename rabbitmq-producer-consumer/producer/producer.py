"""
Producer — publishes N numbered messages to a RabbitMQ queue.

Each message is sent one at a time with a configurable delay so you can
watch the consumer receive them in real time on the other side.

Key concepts shown here:
  - durable queue: survives a RabbitMQ broker restart
  - persistent messages (delivery_mode=2): not lost if the broker restarts
    before the consumer acknowledges them
  - basic_publish with routing_key: direct routing through the default exchange

Environment variables:
  RABBITMQ_HOST     RabbitMQ hostname          (default: localhost)
  RABBITMQ_USER     Username                   (default: guest)
  RABBITMQ_PASS     Password                   (default: guest)
  QUEUE_NAME        Target queue               (default: messages)
  MESSAGE_COUNT     Number of messages to send (default: 20)
  MESSAGE_DELAY     Seconds between messages   (default: 1)
"""

import os
import sys
import time
import datetime
import pika

HOST  = os.getenv("RABBITMQ_HOST",    "localhost")
USER  = os.getenv("RABBITMQ_USER",    "guest")
PASS  = os.getenv("RABBITMQ_PASS",    "guest")
QUEUE = os.getenv("QUEUE_NAME",       "messages")
COUNT = int(os.getenv("MESSAGE_COUNT", "20"))
DELAY = float(os.getenv("MESSAGE_DELAY", "1"))


def now():
    return datetime.datetime.now().strftime("%H:%M:%S")


# ── Connect to broker ─────────────────────────────────────────────────────────
print(f"[{now()}] [producer] Connecting to RabbitMQ at {HOST}:5672 …")
try:
    connection = pika.BlockingConnection(
        pika.ConnectionParameters(
            host=HOST,
            credentials=pika.PlainCredentials(USER, PASS),
        )
    )
except pika.exceptions.AMQPConnectionError as exc:
    print(f"[{now()}] [producer] ERROR: {exc}", file=sys.stderr)
    sys.exit(1)

channel = connection.channel()

# ── Declare queue ─────────────────────────────────────────────────────────────
# durable=True: the queue definition survives a broker restart.
# This is idempotent — safe to call even if the queue already exists.
channel.queue_declare(queue=QUEUE, durable=True)
print(f"[{now()}] [producer] Queue '{QUEUE}' ready  (durable=True)")
print(f"[{now()}] [producer] Sending {COUNT} messages  delay={DELAY}s\n")

# ── Publish messages ──────────────────────────────────────────────────────────
for i in range(1, COUNT + 1):
    body = f"Message {i} of {COUNT}"

    channel.basic_publish(
        exchange="",          # default exchange — routes directly by queue name
        routing_key=QUEUE,    # queue name acts as the routing key
        body=body,
        properties=pika.BasicProperties(
            delivery_mode=2,  # persistent: written to disk, survives broker restart
        ),
    )

    print(f"  [{i:>2}/{COUNT}]  {now()}  →  {body}")
    time.sleep(DELAY)

# ── Done ──────────────────────────────────────────────────────────────────────
connection.close()
print(f"\n[{now()}] [producer] All {COUNT} messages sent. Connection closed.")
