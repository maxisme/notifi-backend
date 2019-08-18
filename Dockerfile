FROM golang:1.12-alpine

USER root

ENV GOROOT /usr/local/go
ENV GOPATH $HOME/go
ENV PATH $PATH:$GOROOT/bin

RUN mkdir /.cache && chmod 777 /.cache
RUN apk update
RUN apk upgrade
RUN apk add git
RUN apk add gcc
RUN apk add libc-dev

RUN go get -u github.com/didip/tollbooth
RUN go get -u github.com/didip/tollbooth/limiter
RUN go get -u github.com/go-sql-driver/mysql
RUN go get -u github.com/gorilla/schema
RUN go get -u github.com/gorilla/websocket
RUN go get -u github.com/satori/go.uuid
RUN go get -u golang.org/x/crypto/bcrypt
RUN go get -u github.com/google/uuid
RUN go get -u github.com/getsentry/sentry-go
RUN go get -u github.com/TV4/graceful

RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go build -o notifi .
CMD ["/app/notifi"]