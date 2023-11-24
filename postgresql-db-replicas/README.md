# PostgreSQL DB Replicas

An example of a PostgreSQL database with multiple read replicas.

## Usage

- Start the containers with `docker-compose up -d` (or `make compose`)
- Fetch list of users with `curl http://localhost:8080`
- Add new user with `curl -X POST http://localhost:8080 -d '{"name": "JohnDoe"}'`
