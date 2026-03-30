build:
	go build -o snapshot .

install: build
	mv snapshot /usr/local/bin/snapshot

uninstall:
	rm -f /usr/local/bin/snapshot

.PHONY: build install uninstall
