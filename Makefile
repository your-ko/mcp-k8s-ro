BINARY := mcp-k8s-ro
MODULE := github.com/your-ko/mcp-k8s-ro

.PHONY: build run test clean tidy

tidy:
	go mod tidy

build:
	go build -ldflags \
    "-X main.GitCommit=test -X main.BuildDate=test -X main.Version=POC" \
    -o bin/mcp-k8s-ro ./cmd/mcp-k8s-ro/

run: build
	./$(BINARY)

test:
	go test ./...

clean:
	rm -f $(BINARY)
