.PHONY: build clean

PROGRAM_NAME=main

build:
	env GOOS=linux \
	GOARCH=amd64 \
	go build -o bin/$(PROGRAM_NAME) main.go

clean:
	rm -rf bin || true
