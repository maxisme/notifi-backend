FROM golang:1.12-alpine

RUN set -ex
RUN apk update
RUN apk add --no-cache git

ADD . /app/
WORKDIR /app
RUN go build -o notifi .
CMD ["/app/notifi"]