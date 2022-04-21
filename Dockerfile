#syntax=docker/dockerfile:1

FROM golang:alpine
#RUN addgroup -g 1000  appgroup && adduser -u 1000 -G appgroup -h /appuser -D appuser
#USER appuser
WORKDIR /appuser

COPY go.mod /appuser
COPY go.sum /appuser
COPY *.go /appuser
#USER root
#RUN chown appuser /appuser	

RUN go build -o log-watcher main.go 
CMD ["./log-watcher"]
