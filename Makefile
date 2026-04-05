BINARY := coderoo
DIST := dist
MODULE := github.com/coderoo-dev/coderoo

.PHONY: build build-all test bench vet clean

build:
	go build -o $(DIST)/$(BINARY) ./cmd/coderoo

build-all:
	GOOS=linux   GOARCH=amd64 go build -o $(DIST)/$(BINARY)-linux-amd64   ./cmd/coderoo
	GOOS=darwin  GOARCH=amd64 go build -o $(DIST)/$(BINARY)-darwin-amd64  ./cmd/coderoo
	GOOS=darwin  GOARCH=arm64 go build -o $(DIST)/$(BINARY)-darwin-arm64  ./cmd/coderoo
	GOOS=windows GOARCH=amd64 go build -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/coderoo

test:
	go test ./...

bench:
	go test -bench=. -benchmem -timeout 300s ./internal/build/

vet:
	go vet ./...

clean:
	rm -rf $(DIST) .cache
