.PHONY: compose shutdown clean client

client:
	$(MAKE) -C $@ build

compose: client
	docker compose up -d

shutdown:
	docker compose down

clean: shutdown
	rm -rf ./tmp
	$(MAKE) -C client clean
