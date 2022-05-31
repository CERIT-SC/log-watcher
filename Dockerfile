#syntax=docker/dockerfile:1

FROM golang:alpine

WORKDIR /app
RUN mkdir /logs

COPY go.mod /app
COPY go.sum /app
COPY *.go /app
COPY log-watcher-starter /app

RUN go build -o log-watcher main.go 
CMD ["./log-watcher-starter"]