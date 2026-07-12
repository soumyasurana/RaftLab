FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/node ./cmd/node

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/node .

# TODO: create data directory if needed

CMD ["./node"]
