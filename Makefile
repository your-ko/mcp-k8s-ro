BINARY := mcp-k8s-ro
MODULE := github.com/your-ko/mcp-k8s-ro
# renovate: datasource=github-releases depName=vektra/mockery versioning=semver
MOCKERY_VERSION=v3.7.0

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

docker-build:
	docker build . -t mcp-k8s-ro

generate-mocks:
	docker run --rm -v "$$PWD:/src" -w /src/ vektra/mockery:${MOCKERY_VERSION}
