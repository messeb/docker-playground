.PHONY: webserver compose shutdown clean

webserver:
	$(MAKE) -C $@ build

compose: webserver
	docker compose up -d

shutdown:
	docker compose down

clean: shutdown
	rm -rf ./webserver/bin
