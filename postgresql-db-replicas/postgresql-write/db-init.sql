\c service;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

INSERT INTO users (name)
VALUES ('username1'), ('username2'), ('username3'), ('username4'), ('username5');
