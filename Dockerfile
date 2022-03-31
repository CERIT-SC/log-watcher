#syntax=docker/dockerfile:1

FROM golang:alpine
#in my opinion too big image...maybe in the future take something else
WORKDIR /app

COPY . /app
#ADD ./pv-cleaner ./
#RUN useradd -u 1000 testuser
#USER testuser	
RUN go build -o log-watcher main.go 
CMD ["/app/log-watcher"]
