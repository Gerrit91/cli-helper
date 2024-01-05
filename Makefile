.PHONY: build
build:
	go build -o bin/cli-helper -ldflags '-extldflags "-static"'
	strip bin/cli-helper

.PHONY: clean
clean:
	rm -f bin/*
