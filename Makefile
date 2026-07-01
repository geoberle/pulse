include .bingo/Variables.mk

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

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run -v ./...
.PHONY: lint

lint-fix: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --fix -v ./...
.PHONY: lint-fix

deepcopy: $(DEEPCOPY_GEN) $(GOIMPORTS)
	DEEPCOPY_GEN=$(DEEPCOPY_GEN) hack/update-deepcopy.sh
	$(GOIMPORTS) -w -local github.com/geoberle/pulse internal/workitem/zz_generated.deepcopy.go
.PHONY: deepcopy

verify-deepcopy: deepcopy
	./hack/verify.sh deepcopy
.PHONY: verify-deepcopy

generate: deepcopy
.PHONY: generate

verify: verify-deepcopy
.PHONY: verify

tidy:
	go mod tidy
.PHONY: tidy

clean:
	rm -f $(BINARY)
.PHONY: clean
