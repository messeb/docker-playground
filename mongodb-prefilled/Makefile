.PHONY: compose shutdown clean

compose:
	docker compose up -d

shutdown:
	docker compose down

clean: shutdown
	rm -rf ./mongodb
