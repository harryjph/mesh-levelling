.PHONY: processor
processor:
	go build --tags wayland --ldflags '-w -s' ./cmd/processor