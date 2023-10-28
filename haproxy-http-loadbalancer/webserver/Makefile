.PHONY: build docker_build clean

PROGRAM_NAME=main

build:
	env GOOS=linux \
	GOARCH=amd64 \
	go build -o bin/$(PROGRAM_NAME) main.go

docker_build: build
	docker build -name webserver -t webserver . 

run: docker_build
	docker run -d --rm --name webserver -p 8080:8080 webserver

clean:
	docker stop webserver || true
	docker rmi -f webserver || true
	rm -rf bin || true
