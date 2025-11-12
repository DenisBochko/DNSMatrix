# Stage 1: Builder
FROM golang:1.25.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go run ./cmd/keygen
RUN GOOS=linux go build -ldflags="-s -w" -o /app/bin/hackathon-back ./cmd/hackathon_back

# Stage 2: Run
FROM ubuntu:latest

WORKDIR /app

COPY --from=builder /app/bin/hackathon-back /app/bin/hackathon-back
COPY --from=builder /app/ecdsa_private.pem /app/ecdsa_private.pem
COPY --from=builder /app/ecdsa_public.pem /app/ecdsa_public.pem
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/geobase /app/geobase

EXPOSE 8080

CMD ["/app/bin/hackathon-back"]
