BINARY=pulse

build:
	go build -o $(BINARY) ./cmd/pulse
.PHONY: build

test:
	go test -timeout 120s -race -cover ./...
.PHONY: test

update-testdata:
	umask 0022 && UPDATE=1 go test ./...
.PHONY: update-testdata

test-compile:
	go test -run='^$$' ./...
.PHONY: test-compile

lint:
	go tool golangci-lint run -v ./...
.PHONY: lint

lint-fix:
	go tool golangci-lint run --fix -v ./...
.PHONY: lint-fix

tidy:
	go mod tidy
.PHONY: tidy

clean:
	rm -f $(BINARY)
.PHONY: clean
