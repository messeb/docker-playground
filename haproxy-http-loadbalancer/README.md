# HAProxy loadbalancer with multiple HTTP web servers

This is a simple example of a load balancer with multiple web servers.

## Usage

- Configure `haproxy/haproxy.cfg` with the aliases of your web servers.
- Adjust the webservers in `docker-compose.yml` to your needs.
- Start the containers with `docker-compose up -d` (or `make compose`)
