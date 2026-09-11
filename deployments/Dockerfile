FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /app/node ./cmd/node
RUN CGO_ENABLED=0 go build -o /app/controller ./cmd/controller

FROM alpine:latest
WORKDIR /app
RUN mkdir -p /app/data
COPY --from=builder /app/node .
COPY --from=builder /app/controller .

CMD ["./node"]
