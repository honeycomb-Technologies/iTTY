.PHONY: build-daemon test-daemon lint-daemon run-daemon clean help

## build-daemon: Build the iTTY desktop daemon
build-daemon:
	$(MAKE) -C daemon build

## test-daemon: Run daemon tests
test-daemon:
	$(MAKE) -C daemon test

## lint-daemon: Lint daemon code
lint-daemon:
	$(MAKE) -C daemon lint

## run-daemon: Build and run the daemon
run-daemon:
	$(MAKE) -C daemon run

## clean: Clean all build artifacts
clean:
	$(MAKE) -C daemon clean

## help: Show available commands
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
