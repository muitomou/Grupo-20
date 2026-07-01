FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/datanode ./datanode
RUN go build -o /bin/gateway ./gateway
RUN go build -o /bin/broker ./broker
RUN go build -o /bin/producer ./producer
RUN go build -o /bin/cliente ./cliente

FROM alpine:latest
WORKDIR /app
COPY --from=builder /bin/* /usr/local/bin/
COPY pedidos.csv ./