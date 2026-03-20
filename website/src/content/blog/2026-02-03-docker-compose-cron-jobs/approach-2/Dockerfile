FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY main.go job.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o app main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o job job.go

FROM alpine:3.23

WORKDIR /app

COPY --from=builder /build/app /app/app
COPY --from=builder /build/job /usr/local/bin/job
COPY crontab /etc/crontabs/root

RUN chmod +x /app/app /usr/local/bin/job
