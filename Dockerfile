FROM golang:1.26 AS builder

ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG VERSION=dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags "-X main.GitCommit=${GIT_COMMIT} \
                -X main.BuildDate=${BUILD_DATE} \
                -X main.Version=${VERSION}" \
      -o bin/mcp-k8s-ro ./cmd/mcp-k8s-ro/


FROM gcr.io/distroless/static:nonroot

LABEL io.modelcontextprotocol.server.name="io.github.your-ko/mcp-k8s-ro"

COPY --from=builder /app/bin/mcp-k8s-ro /mcp-k8s-ro

# KUBECONFIG env var can be set to point to a mounted kubeconfig file.
# Default matches the standard mount path: -v ~/.kube:/home/nonroot/.kube
ENV KUBECONFIG=/home/nonroot/.kube/config

ENTRYPOINT ["/mcp-k8s-ro"]
