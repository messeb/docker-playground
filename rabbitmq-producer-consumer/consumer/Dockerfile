FROM python:3.12-alpine

WORKDIR /app
RUN pip install --no-cache-dir pika
COPY consumer.py .

# -u disables output buffering so each print() appears immediately in docker logs
CMD ["python", "-u", "consumer.py"]
