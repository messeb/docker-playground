# Prefilled MongoDB container

It creates a prefilled MongoDB container. It is based on the official MongoDB image and adds a script that will import a predefined collection into the database.
It can be easily extend with additional collections.

## Usage

- Update the `init/data.json` file with your data.
- Update the `init/mongo-init.js` file with your config:
  - Name of database
  - Name of collection
  - Username for database
  - Password for database
- Start the container with `docker-compose up -d` (or `make compose`)
