# Stage 1: Builder
FROM golang:1.25.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux go build -ldflags="-s -w" -o /app/bin/agent ./cmd/agent

# Stage 2: Run
FROM ubuntu:latest

WORKDIR /app

COPY --from=builder /app/bin/agent /app/bin/agent

RUN apt-get update && apt-get install -y iputils-ping traceroute dnsutils ca-certificates && rm -rf /var/lib/apt/lists/*

EXPOSE 8080

CMD ["/app/bin/agent"]
